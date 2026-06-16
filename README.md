# LANLink

LANLink is a local network file transfer and communication system.

It enables trusted devices to pair, authenticate, and transfer files over a LAN, Wi-Fi network, or mobile hotspot.

The system has three components: a Go backend agent, a Go CLI client, and a React Native / Expo mobile app. WebSocket handles the authenticated control plane. HTTP handles file transfer data.

---

## Components

### Go Agent

The backend agent runs on a Linux/Mac/Windows machine or Termux on Android. It listens for connections, manages pairing, authenticates devices, and receives file uploads.

### Go CLI

A terminal client for pairing, health checks, ping, messaging, and high-speed file transfer. Designed for Termux on Android and desktop terminals.

### React Native Mobile App

An Expo Router app for Android and iOS. Supports QR-code pairing, file selection, upload queue with progress/speed/ETA, and device management.

---

## Features

### Agent

* Device pairing with rotating tokens
* Persistent device storage
* Authenticated WebSocket sessions
* Ping/pong latency measurement
* Direct messaging
* Automatic LAN address detection
* Terminal QR code for mobile pairing
* Configurable transfer workers and chunk size

### CLI

* Pair, health, ping, devices, message commands
* High-speed parallel HTTP file upload
* Raw binary chunk transfer (no base64 overhead)

### Mobile App

* QR code scanning for instant pairing
* Manual address/token pairing
* Multi-file queue with automatic sequential upload
* Live progress, speed (MB/s), and ETA
* Cancel active upload
* Retry failed or cancelled transfers
* Stop all / Start all / Clear completed
* Native streaming upload (no JS memory overhead)
* Disk space check before copying content:// URIs

---

## Architecture

```text
Mobile App / CLI
        │
        ├── WebSocket
        │       ├── Authentication
        │       ├── Ping/Pong
        │       └── Direct Messages
        │
        └── HTTP
                ├── CLI Parallel Chunk Upload
                ├── Mobile Streaming Upload
                └── Resumable Upload
                        │
                        ▼
                  Go Agent (Linux / Termux)
```

* WebSocket is used for the authenticated control plane.
* HTTP is used for all file transfer data.
* CLI uses parallel chunked uploads for maximum throughput.
* Mobile uses a single streaming upload per file via Expo FileSystem native upload task.

---

## API Endpoints

### Health

```text
GET /health
```

Returns agent status.

### Pairing

```text
POST /pair
```

Pair a new device with a pairing token. Returns device ID and auth token.

### Devices

```text
GET /devices
```

List paired devices. Requires `Authorization: Bearer <token>`.

### WebSocket

```text
GET /ws
```

Authenticated WebSocket connection for control messages.

### CLI Chunked Transfer

```text
POST   /transfers/start
PUT    /transfers/{id}/chunks/{index}?offset={offset}
POST   /transfers/{id}/finish
```

Three-stage transfer: start session, upload raw binary chunks in parallel, finalize and save.

### Mobile Streaming Upload

```text
POST /transfers/upload
```

Single-request streaming upload. Headers: `X-Filename`, `X-Transfer-Id`, `Authorization`. Body: raw file bytes.

### Resumable Transfer

```text
POST   /transfers/resumable/start
GET    /transfers/resumable/{id}/status
PUT    /transfers/resumable/{id}/chunk?offset={offset}
POST   /transfers/resumable/{id}/finish
DELETE /transfers/resumable/{id}
```

Offset-based resumable upload. Supports pause at the backend level (state kept in memory).

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

Pairing token: 123456
Use this token to pair a new device.
Scan this QR code from the mobile app to pair:
<QR code>
```

### Pair via CLI

```bash
go run ./cli pair 192.168.1.42:8787 123456
```

### Pair via Mobile

1. Install the app from `mobile/`
2. Enter the agent address manually, or scan the terminal QR code
3. Enter the pairing token
4. Tap "Pair and save"

### Send Files via CLI

```bash
go run ./cli send-file 192.168.1.42:8787 ./large.zip
```

### Send Files via Mobile

1. Open the Device tab
2. Tap "Send file"
3. Select one or more files
4. Monitor progress in the Transfers tab

---

## Build

### Go Agent and CLI

```bash
mkdir -p bin
go build -o bin/agent ./agent
go build -o bin/lanlink ./cli
```

### Mobile App

```bash
cd mobile
npm install
npx expo start
```

Scan the QR code with Expo Go on your device.

---

## Configuration

LANLink can be configured using a `.env` file in the project root.

```env
LANLINK_PORT=8787
TRANSFER_CHUNK_SIZE=1048576
TRANSFER_MAX_IN_FLIGHT_CHUNKS=16
```

| Variable                        | Description                           | Default |
| ------------------------------- | ------------------------------------- | ------- |
| `LANLINK_PORT`                  | Port used by the agent                | `8787`  |
| `TRANSFER_CHUNK_SIZE`           | File chunk size in bytes              | `65536` |
| `TRANSFER_MAX_IN_FLIGHT_CHUNKS` | Number of parallel HTTP chunk uploads | `16`    |

Recommended transfer settings for LAN:

```text
1 MB × 16 workers
2 MB × 16 workers
2 MB × 32 workers
```

---

## Repository Structure

```text
lanlink/
├── agent/                          # Go backend agent
│   ├── main.go                     # Entry point, route registration
│   ├── auth.go                     # Bearer token authentication
│   ├── pair.go                     # Device pairing handler
│   ├── pairing_qr.go               # Terminal QR code generation
│   ├── pairing_state.go            # Pairing manager state
│   ├── health.go                   # Health endpoint
│   ├── devices.go                  # Device list endpoint
│   ├── http_transfer.go            # CLI chunked transfer handlers
│   ├── http_upload.go              # Mobile streaming upload handler
│   ├── http_resumable_transfer.go  # Resumable transfer handlers
│   └── ws/                         # WebSocket handlers
│       ├── handler.go              # Connection upgrade
│       ├── auth.go                 # WebSocket authentication
│       ├── session.go              # Message routing loop
│       ├── messages.go             # Ping, hello, direct message
│       ├── file_transfer.go        # WS file transfer (legacy)
│       └── state.go                # WS transfer manager
│
├── cli/                            # Go CLI client
│   ├── main.go                     # Command routing
│   ├── pair.go                     # Pair command
│   ├── health.go                   # Health command
│   ├── ping.go                     # Ping command
│   ├── devices.go                  # Devices command
│   ├── send_file.go                # Parallel file upload
│   └── ws/                         # WebSocket client
│       ├── client.go               # WS connection
│       ├── auth.go                 # WS auth
│       └── messages.go             # Message handling
│
├── mobile/                         # React Native / Expo app
│   ├── app/                        # Expo Router screens
│   │   ├── _layout.tsx             # Root layout
│   │   ├── index.tsx               # Entry redirect
│   │   ├── setup.tsx               # Agent address entry
│   │   ├── pair.tsx                # Manual + QR pairing
│   │   └── (tabs)/                 # Main tab screens
│   │       ├── _layout.tsx         # Tab bar layout
│   │       ├── device.tsx          # Device info + send file
│   │       ├── transfers.tsx       # Upload queue + progress
│   │       └── settings.tsx        # Agent address + credentials
│   ├── src/
│   │   ├── hooks/                  # React hooks
│   │   ├── lib/                    # Core logic
│   │   │   ├── api/                # HTTP client + endpoints
│   │   │   ├── protocol/           # Message schema + types
│   │   │   ├── socket/             # WebSocket client
│   │   │   ├── storage/            # SecureStore + AsyncStorage
│   │   │   └── transfer/           # Upload manager
│   │   └── store/                  # Zustand stores
│   └── package.json
│
├── internal/                       # Shared Go packages
│   ├── auth/                       # Token generation
│   ├── clientconfig/               # CLI credential storage
│   ├── config/                     # Environment config
│   ├── network/                    # LAN IP detection
│   ├── pairing/                    # Pairing token manager
│   ├── paths/                      # File paths
│   ├── store/                      # Device JSON store
│   ├── transfer/                   # Chunked transfer manager
│   └── wsutil/                     # WebSocket utilities
│
├── protocol/                       # Shared protocol types
│   ├── messages.go                 # Message envelope
│   ├── file_chunk.go               # WS file transfer types
│   └── http_transfer.go            # HTTP transfer types
│
├── go.mod
├── go.sum
└── README.md
```

---

## Mobile App Details

### Tech Stack

* Expo SDK 54
* Expo Router 6
* React Native 0.81
* Zustand (state management)
* TanStack Query (server state)
* expo-camera (QR scanning)
* expo-file-system (native upload)
* expo-document-picker (file selection)
* expo-secure-store (credential storage)

### Upload System

The mobile app uses `FileSystem.createUploadTask` from `expo-file-system/legacy` to stream files directly from the device to the agent via `POST /transfers/upload`. This avoids loading the entire file into JavaScript memory.

For `content://` URIs (from document picker with `copyToCacheDirectory: false`), the app checks available disk space before copying to a temp file, and cleans up after upload.

### Transfer Queue

Files are added to a FIFO queue. One file uploads at a time. When an upload completes, the next waiting file starts automatically.

States: `waiting` → `uploading` → `completed` | `failed` | `cancelled`

---

## Development

```bash
go fmt ./...
go test ./...
```

Build and verify:

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
mobile/node_modules/
*.bin
```

---

## Version

```text
v0.5.0-dev
```

---

## License

MIT License
