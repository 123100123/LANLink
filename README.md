# LANLink

LANLink is a local-network file transfer and communication system. Trusted
devices pair, authenticate, and transfer files over a LAN, Wi-Fi network, or
mobile hotspot — no cloud, no accounts.

- **WebSocket** is the authenticated control plane (auth, pairing, ping, messages).
- **HTTP** is the data plane (all file transfers).

## Components

| Component | What it is |
|-----------|------------|
| **`lanlink`** | The unified, **pure-Go terminal binary**. Runs a receiver (`lanlink receive`), sends files (`lanlink send`), and discovers/auto-connects to receivers (`lanlink scan`). Has **zero dependency on the web UI or mobile app**. |
| **`lanlink-agent`** (`./agent`) | The same receiver **plus the browser dashboard** at `/ui`. The only build that embeds `agent-web`. |
| **`lanlink` CLI** (`./cli`) | The original terminal client, preserved; a thin wrapper over the shared client library. |
| **Mobile app** (`mobile/`) | Expo / React Native app for Android & iOS: QR + network-scan pairing, file upload queue with progress/speed/ETA. |

The desktop core is intentionally a **pure-Go terminal application**; the web
dashboard and the mobile app are optional UI layers on top of it.

## Features

- Device pairing with rotating tokens, or **tokenless auto-connect** on trusted LANs
- **LAN discovery**: receivers broadcast a UDP beacon; `lanlink scan` (desktop) and a
  network-scan screen (mobile) find them and connect with no token
- Parallel chunked uploads (CLI), native streaming uploads (mobile), resumable uploads
- A polished **web dashboard**: live transfers, paired clients, output-folder browser, network scan
- Pure-Go, cross-compiles to Linux / Windows / arm64 with no cgo

## Quick start

### Run a receiver

```bash
go run ./cmd/lanlink receive      # headless, terminal QR + progress
# or, with the web dashboard:
go run ./agent                    # then open http://127.0.0.1:8787/ui
```

### Connect and send (desktop → desktop)

```bash
go run ./cmd/lanlink scan                         # discover + auto-connect (no token)
go run ./cmd/lanlink send 192.168.1.42:8787 ./big.zip
```

Or pair explicitly with a token shown by the receiver:

```bash
go run ./cmd/lanlink pair 192.168.1.42:8787 123456
```

### Mobile

1. `cd mobile && npm install && npx expo start`
2. In the app: **Scan agent QR code**, **Scan network** (tokenless), or enter the address + token.
3. Send files from the Device tab; watch progress in the Transfers tab.

## Build

```bash
go build -o bin/lanlink ./cmd/lanlink   # terminal binary (no agent-web)
go build -o bin/agent   ./agent         # dashboard build
```

Cross-platform release binaries (Linux + Windows, pure Go):

```bash
scripts/build-release.sh                # → release/lanlink-* and lanlink-agent-*
```

See [`docs/release.md`](docs/release.md) for Windows executables and the Android APK / EAS path.

## Configuration

`.env` in the project root (see `.env.example`):

| Variable | Description | Default |
|----------|-------------|---------|
| `LANLINK_PORT` | HTTP/WS port | `8787` |
| `LANLINK_DISCOVERY_PORT` | UDP discovery beacon port | `8788` |
| `LANLINK_RECEIVED_DIR` | Output folder for received files | `received` |
| `TRANSFER_CHUNK_SIZE` | Upload chunk size (bytes) | `65536` |
| `TRANSFER_MAX_IN_FLIGHT_CHUNKS` | Parallel chunk uploads | `16` |

## Security note

`lanlink scan` / mobile "Scan network" use an **open, tokenless** endpoint
(`POST /pair/auto`) so any device on the local network can connect without a
token. This is a deliberate convenience for **trusted LANs**. To require tokens,
run the receiver with `lanlink receive --no-discovery` (disables beacon
advertising) and pair via QR/token only. Dashboard filesystem and scan routes
(`/ui/fs/*`, `/ui/discovery/scan`) are **loopback-only** and never exposed to the LAN.

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — how the system is built (packages, data flow, the pure-Go core / UI split)
- [`docs/protocol.md`](docs/protocol.md) — the HTTP, WebSocket, and discovery wire protocol
- [`docs/development.md`](docs/development.md) — building, running, and testing each component
- [`docs/release.md`](docs/release.md) — producing desktop executables and the Android APK
- [`docs/plan.md`](docs/plan.md) — the prototype-to-release implementation plan

## Version

`v0.5.0-dev` · MIT License
