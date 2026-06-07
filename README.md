# LANLink

LANLink is a local network device-control framework written in Go.

It enables trusted devices to pair, authenticate, and communicate over a LAN, Wi-Fi network, or mobile hotspot using a custom JSON protocol over WebSockets.

The project is designed as a learning-focused networking platform and a foundation for future device control features such as file sharing, media control, and mobile applications.

---

## Features

### Implemented

* Device pairing
* Persistent authentication
* HTTP health endpoint
* Authenticated WebSocket sessions
* Ping/Pong latency measurement
* Direct messaging
* Persistent device storage
* Automatic LAN address detection

### Planned

* File transfer
* Media control
* Volume control
* LAN discovery
* React Native mobile client
* Reliability and reconnect logic

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
      ├── Media Control
      └── Volume Control
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
v0.2.0
```

---

## License

MIT License
