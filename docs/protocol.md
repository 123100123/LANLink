# LANLink protocol

The wire formats spoken between clients and a receiver. Go DTOs live in
`protocol/`; the mobile TypeScript mirrors are in
`mobile/src/lib/protocol/payloads.ts`.

Two planes: **HTTP** for data + simple control, **WebSocket** for the
authenticated session. Default port: HTTP/WS `8787`; discovery is mDNS
(`_lanlink._tcp`).

## Authentication

After pairing, a client holds an `auth_token`. Authenticated HTTP requests send:

```
Authorization: Bearer <auth_token>
```

Tokens are validated against the receiver's device store (`internal/store`,
persisted at `~/.lanlink/devices.json`).

## HTTP endpoints

### Health — `GET /health`

Open (no auth). Returns `{ "status": "ok", "service": "lanlink-agent" }`.

### Pairing — `POST /pair`

Open. Body `{ "device_name": string, "token": string }`. Validates the rotating
pairing token; on success returns
`{ "status": "paired", "device_id": string, "auth_token": string }` and rotates
the token. Errors return `{ "status": "error", "error": string }`.

### Tokenless auto-connect — `POST /pair/auto`

Open, **LAN-facing by design**. Body `{ "device_name": string }` (any token is
ignored). Issues credentials **without** a pairing token and returns the same
shape as `/pair`. Used by `lanlink scan` and the mobile network scan. Disable
beacon advertising with `lanlink receive --no-discovery` if you don't want this
discoverable. The `/pair` token flow is unaffected.

### Devices — `GET /devices`

Auth required. Returns `{ "devices": [{ "device_id", "device_name" }] }`.

### Chunked transfer (CLI / desktop)

Auth required. High-throughput parallel upload:

```
POST /transfers/start                         {transfer_id, filename, size}
PUT  /transfers/{id}/chunks/{index}?offset=N  body: raw chunk bytes
POST /transfers/{id}/finish
```

`start` registers the transfer; each `chunks` PUT writes raw bytes at `offset`
(responses: `chunk.received` or `chunk.duplicate`); `finish` verifies the total
size, fsyncs, and renames into the output folder.

### Streaming upload (mobile)

```
POST /transfers/upload
Headers: Authorization, X-Filename, X-Transfer-Id
Body: raw file bytes
```

Single-request streaming upload. Response: `{ "status": "saved", "path": ... }`.

### Resumable transfer

```
POST   /transfers/resumable/start              {transfer_id, filename, size}
GET    /transfers/resumable/{id}/status
PUT    /transfers/resumable/{id}/chunk?offset=N
POST   /transfers/resumable/{id}/finish
DELETE /transfers/resumable/{id}
```

Offset-based; the server tracks received bytes so an interrupted upload can
resume from `status`.

### Dashboard routes (`/ui/*`) — loopback only

Served only by the `lanlink-agent` binary and rejected for non-loopback callers:

```
GET  /ui                         dashboard HTML
GET  /ui/state                   JSON snapshot (polled ~1/s)
GET  /ui/qr                      pairing QR PNG
GET  /ui/fs/list?path=…          list directories (folder browser)
POST /ui/fs/mkdir                {path, name} — create a folder
GET  /ui/discovery/scan          discover other receivers (self filtered)
GET/POST /ui/settings/output-dir[/reset]
POST /ui/clients/unpair          {device_id}
POST /ui/transfers/cancel        {transfer_id}
```

## WebSocket (`GET /ws`)

Messages use a JSON envelope (`protocol/message.go`):

```json
{ "type": string, "id": string, "timestamp": number, "payload": <type-specific> }
```

Flow:

1. **auth** — client sends `{type:"auth", payload:{token}}`; server replies
   `auth.success` (`{device_id, device_name}`) or `auth.failed` (`{error}`).
2. **ping / pong** — `{type:"ping", payload:{sent_at}}` → `{type:"pong",
   payload:{sent_at, received_at}}`; the client computes round-trip latency.
3. **hello** — debug echo.
4. **direct_message** — `{type:"direct_message", payload:{text}}` →
   `direct_message.response` (`{status}`).

## Discovery (mDNS / DNS-SD)

Receivers advertise an **`_lanlink._tcp`** service over mDNS (`internal/discovery`,
via `grandcat/zeroconf`). The instance name is the hostname; the SRV record
carries the receiver's HTTP port; TXT records carry extra info:

```
_lanlink._tcp.local   port=8787   TXT: v=0.5.0-dev  open=true
```

`Scan` browses the service and dedupes by resolved `ip:port`. `open=true` means
the receiver accepts tokenless `/pair/auto`. The mobile app currently discovers
by sweeping the `/24` for `GET /health` (an active probe, independent of the
advertisement); mDNS browsing on mobile is planned alongside the phone receiver.

## Pairing QR payload

The terminal/dashboard QR encodes:

```json
{ "t": "l", "a": "<ip:port>", "tk": "<pairing token>" }
```

The mobile pair screen parses this (`t === "l"`) to prefill address + token.
