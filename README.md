# LANLink

LANLink is a local network communication framework written in Go.

It enables trusted devices to pair, authenticate, and communicate over a LAN, Wi-Fi network, or mobile hotspot.

The project uses WebSockets for authenticated control messages and HTTP for high-throughput file transfer. It is designed as a learning-focused networking platform and a foundation for building secure local device-to-device communication systems.

---

## Features

### Implemented

* Device pairing
* Persistent authentication
* HTTP health endpoint
* Authenticated WebSocket sessions
* Ping/Pong latency measurement
* Direct messaging
* High-speed HTTP file transfer
* Chunked large-file transfer
* Parallel chunk uploads
* Upload progress tracking
* Safe file storage with overwrite protection
* Persistent device storage
* Automatic LAN address detection
* Configurable transfer chunk size
* Configurable parallel transfer workers

### Planned

* Reliability and reconnect logic
* LAN discovery
* Device management
* Transfer retry/resume support
* Advanced security features
* Improved transfer benchmarking tools

---

## Architecture

```text
CLI / Remote Device
        │
        ├── WebSocket
        │       ├── Authentication
        │       ├── Ping/Pong
        │       └── Direct Messages
        │
        └── HTTP
                └── Parallel File Transfer
                        │
                        ▼
                  Linux Agent
```

LANLink separates control messages from file transfer data:

* WebSocket is used for authenticated communication, ping/pong, and direct messages.
* HTTP is used for raw binary file chunks.
* File chunks are uploaded in parallel using multiple HTTP workers.
* The agent writes chunks safely using offsets and protects existing files from being overwritten.

---

## Quick Start

### Start Agent

```bash
go run ./agent
```

Example output:

```text
LANLink agent listening on :8787

Available addresses:
127.0.0.1:8787
192.168.1.42:8787
```

Use the LAN address when connecting from another device on the same network.

---

### Pair Device

```bash
go run ./cli pair 192.168.1.42:8787 123456
```

Pairing stores local credentials so future commands can authenticate automatically.

---

### Check Health

```bash
go run ./cli health 192.168.1.42:8787
```

---

### List Devices

```bash
go run ./cli devices 192.168.1.42:8787
```

---

### Ping Agent

```bash
go run ./cli ping 192.168.1.42:8787
```

---

### Send Message

```bash
go run ./cli message 192.168.1.42:8787 "hello from termux"
```

---

### Transfer Files

```bash
go run ./cli send-file 192.168.1.42:8787 ./large.zip
```

The `send-file` command uses the high-speed HTTP transfer path. Files are split into chunks, uploaded in parallel, and reassembled by the agent.

---

## Build

Build the agent and CLI:

```bash
mkdir -p bin
go build -o bin/agent ./agent
go build -o bin/lanlink ./cli
```

Run the agent:

```bash
./bin/agent
```

Use the CLI:

```bash
./bin/lanlink health 192.168.1.42:8787
./bin/lanlink pair 192.168.1.42:8787 123456
./bin/lanlink send-file 192.168.1.42:8787 ./large.zip
```

---

## Configuration

LANLink can be configured using a `.env` file.

Example:

```env
LANLINK_PORT=8787
TRANSFER_CHUNK_SIZE=524288
TRANSFER_MAX_IN_FLIGHT_CHUNKS=16
```

### Options

| Variable                        | Description                           | Default |
| ------------------------------- | ------------------------------------- | ------- |
| `LANLINK_PORT`                  | Port used by the agent                | `8787`  |
| `TRANSFER_CHUNK_SIZE`           | File chunk size in bytes              | `65536` |
| `TRANSFER_MAX_IN_FLIGHT_CHUNKS` | Number of parallel HTTP chunk uploads | `16`    |

Recommended transfer settings:

```text
512 KB × 16 workers
1 MB × 16 workers
1 MB × 32 workers
2 MB × 16 workers
2 MB × 32 workers
```

Example:

```env
TRANSFER_CHUNK_SIZE=1048576
TRANSFER_MAX_IN_FLIGHT_CHUNKS=32
```

Very high worker counts may reduce performance on some systems because of CPU usage, memory pressure, socket overhead, router limits, or storage speed.

---

## File Transfer Design

LANLink file transfer uses three HTTP stages:

```text
POST /transfers/start
PUT  /transfers/{id}/chunks/{index}?offset={offset}
POST /transfers/{id}/finish
```

The transfer flow is:

1. The CLI starts a transfer session.
2. The CLI splits the file into chunks.
3. Multiple workers upload chunks in parallel.
4. The agent writes each chunk at its correct file offset.
5. The CLI asks the agent to finalize the transfer.
6. The agent verifies the received size and safely stores the final file.

This avoids base64 encoding and JSON payload overhead for file data. File bytes are sent as raw HTTP request bodies.

---

## Repository Structure

```text
lanlink/

├── agent/
│   ├── main.go
│   ├── http_transfer.go
│   └── ws/
│
├── cli/
│   ├── main.go
│   └── send_file.go
│
├── internal/
│   ├── auth/
│   ├── clientconfig/
│   ├── config/
│   ├── network/
│   ├── paths/
│   ├── store/
│   ├── transfer/
│   └── wsutil/
│
├── protocol/
├── go.mod
├── go.sum
└── README.md
```

---

## Development Notes

Run formatting and tests before committing:

```bash
go fmt ./...
go test ./...
```

Build both binaries:

```bash
mkdir -p bin
go build -o bin/agent ./agent
go build -o bin/lanlink ./cli
```

Generated files and runtime data should not be committed:

```text
.env
bin/
data/
received/
testdata/
*.bin
```

---

## Version

Current version:

```text
v0.4.0-dev
```

---

## License

MIT License
