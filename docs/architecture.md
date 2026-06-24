# LANLink architecture

This document explains how LANLink is put together for developers reading or
extending the code. For the exact wire formats see [`protocol.md`](protocol.md);
for build/run/test workflows see [`development.md`](development.md).

## Big picture

LANLink has two communication planes between any client and a receiver:

- **Control plane — WebSocket (`/ws`)**: authentication, ping/pong, direct messages.
- **Data plane — HTTP**: all file transfers (chunked, streaming, resumable).

There are three runtimes: the **desktop binaries** (Go), the **web dashboard**
(vanilla JS embedded in one of those binaries), and the **mobile app** (Expo).

### The defining constraint: a pure-Go terminal core

The desktop core is a **pure-Go terminal application with zero dependency on
`mobile/` or `agent-web/`.** Two binaries are built from the same Go source:

| Binary | Package | Imports `agent-web`? |
|--------|---------|----------------------|
| `lanlink` | `cmd/lanlink` | **No** — verified by `go list -deps ./cmd/lanlink` |
| `lanlink-agent` | `agent` | Yes — it embeds and serves the dashboard |

`lanlink receive` and `lanlink-agent` run the **same receiver core**
(`internal/agentserver`); the agent binary just layers the web UI on top via an
injectable route hook. This keeps the terminal product minimal and lets the
dashboard evolve independently.

## Package layout

```
cmd/lanlink/            Unified terminal binary: subcommand router (receive/send/scan/pair/…).
                        Imports internal/agentserver + internal/client. No agent-web.

agent/                  Dashboard binary. main.go is a thin wrapper that calls
  main.go               agentserver.Run() with a RegisterRoutes hook for the dashboard.
  dashboard/            Web layer: HTTP routes for /ui/*, QR image, folder browser,
                        network scan. Reads core state via internal/agentserver. Embeds agent-web.
  ws/                   WebSocket upgrade, auth, session loop, message handlers.

agent-web/              Dashboard frontend (index.html, assets/app.js, assets/styles.css),
                        embedded via go:embed in embed.go. Vanilla HTML/CSS/JS, no build step.

internal/
  agentserver/          THE PURE-GO RECEIVER CORE. HTTP handlers (pair, pair_auto, devices,
                        health, http_transfer, http_upload, http_resumable_transfer), the
                        transfer-state registry (state.go), output-folder settings (settings.go),
                        terminal QR + progress, the discovery beacon, and Run(Options).
  client/               Shared client operations used by cmd/lanlink and cli: health, pair,
                        devices, ping, message, send (commands.go), ws (ws.go), scan + pairAuto (scan.go).
  discovery/            UDP beacon Announce + Scan; broadcast_unix.go / broadcast_windows.go set
                        SO_BROADCAST (build-tagged, stdlib syscall only).
  transfer/             manager.go = receiver-side chunk writer (used by the core);
                        sender.go = reusable parallel chunked uploader + TerminalProgress().
  auth/ clientconfig/ config/ network/ pairing/ paths/ store/ wsutil/   small shared helpers.

protocol/               Shared DTOs (pure data): pairing, devices, health, ping, message,
                        http_transfer, file_chunk, message envelope.

cli/                    Original terminal client, preserved. A thin wrapper over internal/client.

mobile/                 Expo / React Native app (see "Mobile" below).
```

### Why the `agent/dashboard` ↔ `internal/agentserver` split exists

Originally `agent/dashboard` held both the web UI **and** the transfer-state
registry — and the terminal progress display read from it, so the terminal path
transitively depended on `agent-web`. The refactor moved the **state registry**
(`AddTransfer`, `UpdateTransfer`, `GetState`, output-folder settings, paired
clients, cancellation) into `internal/agentserver`. Now:

- The core owns state; `agent/dashboard` is a thin web layer that **reads** core
  state and serves `/ui`.
- Dependencies flow one way: `agent/dashboard → internal/agentserver` (never back).
- `cmd/lanlink` imports only the core, so it carries no web UI.

## How a file transfer flows (receiver side)

All three upload paths share the same shape: authenticate, write to a temp file,
update the state registry, then atomically rename into the output folder.

1. **CLI / desktop chunked** (`/transfers/start` → `PUT .../chunks/{i}` → `/finish`):
   `internal/transfer.Manager` allocates a temp file, `WriteChunk` writes at each
   offset (deduping retransmits), `Finalize` verifies size, fsyncs, and renames.
   Handlers in `agentserver/http_transfer.go` mirror progress into the state
   registry (`AddTransfer`/`UpdateTransfer`/`CompleteTransfer`).
2. **Mobile streaming** (`POST /transfers/upload`): a single request body is
   streamed to a temp file in `agentserver/http_upload.go`, updating progress
   every 500 ms, then renamed.
3. **Resumable** (`/transfers/resumable/*`): offset-based, state kept in memory
   in `agentserver/http_resumable_transfer.go`; honors the configured output folder.

The **sender** side (desktop) lives in `internal/transfer/sender.go`:
`SendFile(address, token, path, opts)` opens the file, starts a transfer, fans
out chunks across a worker pool, and reports progress through a `ProgressFunc`.
Both `cmd/lanlink send` and `cli send-file` call it via `internal/client`.

## Discovery and tokenless auto-connect

`internal/discovery` is pure Go (stdlib only):

- **`Announce(ctx, info, port)`** opens a UDP socket with `SO_BROADCAST` and
  broadcasts a small JSON `Announcement` (`{service,name,addr,port,v,open}`)
  every ~2s to the global broadcast address and each interface's directed
  broadcast. Receivers start this in `agentserver.Run` unless `--no-discovery`.
- **`Scan(timeout, port)`** binds the discovery port, collects deduped
  announcements, and returns them.
- Cross-platform `SO_BROADCAST` is set in `broadcast_unix.go` /
  `broadcast_windows.go` (build-tagged) using only the standard `syscall` package.

**Tokenless auto-connect**: `POST /pair/auto` (in `agentserver/pair_auto.go`)
issues credentials without validating a token. It's LAN-facing by design — the
deliberate trust relaxation that lets `scan` connect with no token. The token +
QR flow at `/pair` is unchanged, and auto-connected clients appear in the
dashboard with unpair.

Mobile cannot listen for the UDP beacon in managed Expo without a native module,
so the mobile app discovers via an **HTTP `/health` subnet sweep**
(`mobile/src/lib/discovery/sweep.ts`) and then calls `/pair/auto`.

## The web dashboard

`agent-web` is intentionally framework-free vanilla HTML/CSS/JS so it embeds
cleanly into a single binary with no build step. `app.js` polls `/ui/state`
every second and re-renders. The backend (`agent/dashboard`) exposes JSON routes
under `/ui/*`, all gated to loopback by `IsLocalRequest` — filesystem browsing
(`/ui/fs/list`, `/ui/fs/mkdir`) and network scan (`/ui/discovery/scan`) are never
reachable from the LAN.

## Mobile

Expo Router app under `mobile/`:

- **State**: Zustand (`src/store/sessionStore.ts` for credentials,
  `transferStore.ts` for the upload queue) + TanStack Query for device lists.
- **Networking**: `src/lib/api/` (HTTP), `src/lib/socket/lanlinkSocket.ts` (WS),
  `src/lib/protocol/` (Zod-validated envelopes), `src/lib/transfer/` (native
  streaming upload queue), `src/lib/discovery/sweep.ts` (network scan).
- **Screens**: `app/pair.tsx` (QR / network-scan / manual), `app/scan.tsx`
  (subnet sweep + tokenless connect), and the `(tabs)/` device, transfers, and
  settings screens.
- The TS protocol types in `src/lib/protocol/payloads.ts` mirror the Go
  `protocol/` package.

## Conventions

- Go: handlers are `xHandler(w, r)`; packages are short and lowercase; errors are
  returned explicitly; shared state is guarded by `sync.Mutex`. The terminal core
  must never import `agent-web` (enforce with `go list -deps ./cmd/lanlink`).
- Keep reusable logic in `internal/`; keep `main` packages thin.
- Dashboard frontend stays vanilla (no framework/build step).
- Mobile stays Expo-Go / managed-workflow safe (no custom native modules).
