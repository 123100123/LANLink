# LANLink

LANLink is a local network communication framework written in Go.

It enables trusted devices to pair, authenticate, and communicate over a LAN, Wi-Fi network, or mobile hotspot using a custom JSON protocol over WebSockets.

The project is designed as a learning-focused networking platform and a foundation for building secure local device-to-device communication systems.

---

## Features

### Implemented

* Device pairing
* Persistent authentication
* HTTP health endpoint
* Authenticated WebSocket sessions
* Ping/Pong latency measurement
* Direct messaging
* Small file transfer
* Chunked large-file transfer
* Upload progress tracking
* Safe file storage with overwrite protection
* Persistent device storage
* Automatic LAN address detection

### Planned

* Reliability and reconnect logic
* LAN discovery
* Device management
* React Native mobile client
* Advanced security features

---

## Architecture

```text
CLI / Termux
      │
      ▼
Authenticated WebSocket
      │
      ▼
Linux Agent
      │
      ├── Ping/Pong
      ├── Direct Messages
      ├── File Transfer
      └── Future Modules
```

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

### Pair Device

```bash
go run ./cli pair 192.168.1.42:8787 123456
```

### Ping Agent

```bash
go run ./cli ping 192.168.1.42:8787
```

### Send Message

```bash
go run ./cli message 192.168.1.42:8787 "hello from termux"
```

### Transfer Small Files

```bash
go run ./cli send-file 192.168.1.42:8787 ./test.txt
```

### Transfer Large Files

```bash
go run ./cli send-file-chunked 192.168.1.42:8787 ./large.zip
```

---

## Repository Structure

```text
lanlink/

├── agent/
├── cli/
├── internal/
├── protocol/
└── docs/
```

---

## Version

Current version:

```text
v0.3.0
```

## License

MIT License
