# LANLink development guide

How to build, run, and test each component while working on the code. For the
overall design see [`architecture.md`](architecture.md).

## Prerequisites

- Go 1.24+
- Node 18+ and npm (for the mobile app)
- Expo Go on a phone, or an Android emulator, for mobile testing

## Layout recap

- `cmd/lanlink` — the unified terminal (cmd) binary `lanlink` (no web UI)
- `agent` — the receiver-UI build `lanlinkAgent` (embeds `agent-web`)
- `internal/agentserver` — the pure-Go receiver core (shared by both)
- `internal/client`, `internal/transfer` — shared logic
- `cli` — the preserved original CLI
- `mobile` — the Expo app

## Build & run (desktop)

```bash
# Build both binaries
go build -o bin/lanlink      ./cmd/lanlink   # cmd
go build -o bin/lanlinkAgent ./agent         # receiver UI

# Run a headless receiver (terminal QR + progress)
go run ./cmd/lanlink receive
# Run the receiver WITH the dashboard
go run ./agent            # http://127.0.0.1:8787/ui

# Client operations
go run ./cmd/lanlink pair <host:port> <token>
go run ./cmd/lanlink send <host:port> <file>
go run ./cmd/lanlink health <host:port>
```

Useful environment variables (or a `.env`, see `.env.example`):
`LANLINK_PORT`, `LANLINK_HOST`, `LANLINK_RECEIVED_DIR`,
`TRANSFER_CHUNK_SIZE`, `TRANSFER_MAX_IN_FLIGHT_CHUNKS`.

## Verifying the Go code

```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

### The pure-Go-core invariant

The terminal binary must never depend on the web UI. Enforce it:

```bash
go list -deps ./cmd/lanlink | grep -c agent-web   # must print 0
go list -deps ./agent       | grep -c agent-web   # prints 1
```

If a change makes the first command print non-zero, something in the import
chain pulled in `agent-web` — move that dependency behind the dashboard binary.

### Cross-compilation

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./cmd/lanlink
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build ./cmd/lanlink
```

## Manual end-to-end check (desktop)

Isolate credentials so you don't touch your real `~/.lanlink`:

```bash
export HOME=$(mktemp -d)
LANLINK_PORT=8810 LANLINK_RECEIVED_DIR=$(mktemp -d) ./bin/lanlink receive &
# pair with the token the receiver printed, then send
./bin/lanlink pair 127.0.0.1:8810 <token>
./bin/lanlink send 127.0.0.1:8810 ./somefile
```

For the dashboard, run `./bin/lanlinkAgent`, open `http://127.0.0.1:8787/ui`, and check:
QR renders, folder browser lists/creates dirs, a transfer shows live progress,
paired-clients list and cancel/unpair work.

## Mobile

```bash
cd mobile
npm install
npm run typecheck      # tsc --noEmit — run this before committing
npx expo start         # scan the QR with Expo Go
```

Mobile maps to the agent over the LAN — the phone and the agent host must be on
the same network. Pair by scanning the agent's QR code or entering its address +
token.

There are currently no automated mobile UI tests; `npm run typecheck` is the
gate. Manual device testing requires a phone/emulator on the same Wi-Fi.

## Conventions when adding code

- Put reusable logic in `internal/`; keep `main` packages thin.
- New shared client operations go in `internal/client` (both `cmd/lanlink` and
  `cli` should call them — don't duplicate).
- New receiver endpoints go in `internal/agentserver` and are registered in
  `run.go`. Dashboard-only (UI) endpoints go in `agent/dashboard` and must stay
  loopback-only via `IsLocalRequest`.
- Shared DTOs go in `protocol/`; mirror them in `mobile/src/lib/protocol/`.
- Keep `agent-web` framework-free. The mobile app uses a local native module
  (`modules/lanlink-uploader`), so test it in a dev/EAS build, not Expo Go.
- Commit small, build-clean changes; never commit a state where `go build ./...`
  fails.
