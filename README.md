# LANLink

LANLink is a local-network file transfer and communication system. Trusted
devices pair, authenticate, and transfer files over a LAN, Wi-Fi network, or
mobile hotspot — no cloud, no accounts.

- **WebSocket** is the authenticated control plane (auth, pairing, ping, messages).
- **HTTP** is the data plane (all file transfers).

## Components

| Component | What it is |
|-----------|------------|
| **`lanlink`** (cmd) | The unified, **pure-Go terminal binary** (ships as `lanlink` / `lanlink.exe`). Runs a receiver (`lanlink receive`) and sends files (`lanlink send`). Has **zero dependency on the web UI or mobile app**. |
| **`lanlinkAgent`** (receiver UI) | The same receiver **plus the browser dashboard** at `/ui` (`./agent` → `lanlinkAgent-<os>-<arch>`). The only build that embeds `agent-web`. |
| **legacy CLI** (`./cli`) | The original terminal client, preserved for reference; superseded by `lanlink` above. |
| **Mobile app** (`mobile/`) | Expo / React Native app for Android & iOS: QR or address+token pairing, file upload queue with progress/speed/ETA. |

The desktop core is intentionally a **pure-Go terminal application**; the web
dashboard and the mobile app are optional UI layers on top of it.

## Features

- Device pairing with rotating tokens via the receiver's QR code or a manual address + token
- Parallel chunked uploads (CLI), native streaming uploads (mobile), resumable uploads
- A polished **web dashboard**: live transfers, paired clients, output-folder browser
- Pure-Go, cross-compiles to Linux / Windows / arm64 with no cgo

## Quick start

### Run a receiver

```bash
go run ./cmd/lanlink receive      # headless, terminal QR + progress
# or, with the web dashboard:
go run ./agent                    # then open http://127.0.0.1:8787/ui
```

> **Windows receiver:** allow LANLink through Windows Defender Firewall (Private
> networks) so other devices can connect — click **Allow access** on the prompt,
> or, as Administrator, run `netsh advfirewall firewall add rule name="LANLink"
> dir=in action=allow protocol=TCP localport=8787`.

### Connect and send (desktop → desktop)

Pair with the token the receiver prints, then send:

```bash
go run ./cmd/lanlink pair 192.168.1.42:8787 123456
go run ./cmd/lanlink send 192.168.1.42:8787 ./big.zip
```

### Mobile

1. `cd mobile && npm install && npx expo start`
2. In the app: **scan the agent's QR code**, or enter the address + token.
3. Send files from the Device tab; watch progress in the Transfers tab.

## Build

```bash
go build -o bin/lanlink      ./cmd/lanlink   # terminal (cmd) binary (no agent-web)
go build -o bin/lanlinkAgent ./agent         # receiver-UI build (dashboard)
```

Cross-platform release binaries (Linux + Windows, pure Go):

```bash
scripts/build-release.sh                # → release/lanlink[.exe] (cmd) + lanlinkAgent-<os>-<arch> (receiver UI)
```

See [`docs/release.md`](docs/release.md) for Windows executables and the Android APK / EAS path.

### Download & run (terminal binary)

The terminal build ships as a single executable named `lanlink` (Linux) /
`lanlink.exe` (Windows), so it runs directly:

```bash
# Linux: make it executable once, then run
chmod +x lanlink
./lanlink receive
./lanlink pair 192.168.1.42:8787 123456
./lanlink send 192.168.1.42:8787 ./big.zip
```

```powershell
# Windows
.\lanlink.exe receive
.\lanlink.exe send 192.168.1.42:8787 .\big.zip
```

## Configuration

`.env` in the project root (see `.env.example`):

| Variable | Description | Default |
|----------|-------------|---------|
| `LANLINK_PORT` | HTTP/WS port | `8787` |
| `LANLINK_HOST` | Force the advertised LAN address (else auto-detected) | _(auto)_ |
| `LANLINK_RECEIVED_DIR` | Output folder for received files | `received` |
| `TRANSFER_CHUNK_SIZE` | Upload chunk size (bytes) | `65536` |
| `TRANSFER_MAX_IN_FLIGHT_CHUNKS` | Parallel chunk uploads | `16` |

## Security note

Pairing always requires the receiver's token (shown as a QR code or printed on
start), so only devices you explicitly pair can connect, and the receiver rotates
its token after each successful pairing. Dashboard filesystem routes (`/ui/fs/*`)
are **loopback-only** and never exposed to LAN clients.

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — how the system is built (packages, data flow, the pure-Go core / UI split)
- [`docs/protocol.md`](docs/protocol.md) — the HTTP and WebSocket wire protocol
- [`docs/development.md`](docs/development.md) — building, running, and testing each component
- [`docs/release.md`](docs/release.md) — producing desktop executables and the Android APK

## Version

`v1.1.0` · MIT License
