# LANLink: prototype → release-ready — implementation plan

Branch: `ui-update` · Auto mode, continuous, phase-by-phase · No remote pushes · Small verified
commits per phase. The user's `/goal` directive is the authoritative spec; this file mirrors it.

## Core product principle (defining constraint)

**LANLink's core product is a pure-Go terminal application with ZERO dependency on `mobile/` or
`agent-web/`.** Terminal binary `lanlink` builds/runs as pure Go and provides pair, scan, send,
receive entirely from the terminal. Mobile and the web dashboard are optional, separate UI layers.
UI code (`agent-web/`, `mobile/`) may change freely; terminal/core code stays structurally stable —
refactor/fix only where suboptimal or buggy (and actively look for such problems).

Verifiable checks: `go build ./cmd/lanlink` pure Go; `go list -deps ./cmd/lanlink` has **no**
`agent-web`; `lanlink receive` runs headless (terminal QR + progress) and accepts uploads.

### Reconciliation note
`/goal` Phase 3 says "verify `lanlink receive` serves dashboard," which conflicts with the explicit
"fully separate binary / terminal has zero agent-web" decision. **Resolution:** `cmd/lanlink` stays
agent-web-free (primary product); the dashboard is served by the preserved `agent` binary (the only
artifact importing `agent-web`). `internal/agentserver` exposes an injectable extra-route hook, so a
dashboard-enabled `lanlink` build is trivial later. The "serves dashboard + accepts mobile upload"
checks run against the `agent` binary.

## Current architecture (verified) — the key coupling

Module `github.com/123100123/lanlink`, Go 1.24.4, pure-Go deps (cross-compiles). WS = control plane,
HTTP = data plane. **`agent/dashboard` is doing double duty:** (a) the real **transfer-state
registry** (`AddTransfer/UpdateTransfer/Complete/Fail/IsTransferCancelled/ReserveTransferID/
GetActiveTransfers/GetRecentlyCompleted/GetOutputDir/InitSettings/SetAddress/SetToken/
AddPairedClient`) used by core handlers (`agent/http_transfer.go`, `http_upload.go`, `pair.go`) **and
by `agent/terminal_progress.go`**; (b) the **web UI** (`routes/qr/browser/settings.go` + the only
`agent-web` import at `agent/dashboard/routes.go:10`). Splitting (a) into the pure-Go core and leaving
(b) as a thin web layer is what frees the terminal from agent-web.

Other: receiver manager shared (`internal/transfer/manager.go`); sender monolithic in
`cli/send_file.go` (468 LoC); agent is `package main`; no Go tests; no folder browser; routes on the
global mux in `agent/main.go`.

## Decisions
- Terminal binary `cmd/lanlink` (pure Go, no agent-web): `receive`, `send`, `scan`, `pair`, `health`,
  `devices`, `ping`, `message`, `--help`. Primary release artifact.
- Dashboard = the preserved `agent` binary (core + `agent/dashboard` web layer + agent-web embed).
- Transfer-state registry moves into the pure-Go core; `agent/dashboard` becomes a thin read/serve
  web layer over core state (web → core only; no cycle).
- Vanilla HTML/CSS/JS dashboard; folder browser (new) loopback-only.
- LAN discovery = UDP broadcast beacon (desktop) + tokenless open auto-connect (`POST /pair/auto`),
  in the pure-Go core. Mobile discovers via HTTP `/health` subnet sweep (no native module).

## Risks & constraints
- **R1 — core/dashboard split is the central refactor** (move state out of `agent/dashboard/state.go`
  to `internal/agentserver`; repoint handlers + `terminal_progress.go`; verify no agent-web in
  `go list -deps ./cmd/lanlink`). Move state first, then carve the web layer.
- **R2 — loopback-only** for `/ui/fs/list`, `/ui/fs/mkdir`, settings, and `/ui/discovery/scan` (reject
  non-loopback `RemoteAddr`); dashboard-binary only.
- **R3 — don't break mobile** (pairing, `/transfers/upload`, WS protocol byte-compatible; additive only).
- **R4 — sender extraction** keeps `send-file`/`send` behavior identical (progress callback).
- **R5 — pure-Go cross-compile** (`CGO_ENABLED=0`); no machine-specific paths; output dir configurable.
- **R6 — no remote pushes.**
- **R7 — `POST /pair/auto`** is a deliberate LAN-only trust relaxation; clients listed in dashboard
  with unpair; QR/token path intact; document tradeoff.
- **R8 — mobile uses HTTP `/health` sweep** (no `react-native-udp`; Expo-Go/APK safe); desktop uses the beacon.
- **R9 — UDP broadcast quirks**: per-interface broadcast, dedupe, scan timeout + graceful empty.
- **R10 — bug-audit license**: while refactoring terminal/transfer code, fix bugs/races/leaks in scoped commits.

## Target layout
`internal/agentserver/` pure-Go core (receiver HTTP/WS, state registry, terminal QR+progress,
discovery announce, `Run(cfg, opts)` with an extra-route hook) — no agent-web.
`cmd/lanlink/` terminal binary (core + send + scan + pair…), no agent-web.
`agent/` dashboard binary (thin main: core + `agent/dashboard`). `agent/dashboard/` web layer only.
`internal/discovery/` UDP beacon + scan. `internal/transfer/` `sender.go` + `manager.go`.

## Phases (matching `/goal`; each ends in its own verified commit)

**Phase 1 — Baseline verification. (DONE)** `ui-update` confirmed; `go build/vet/test` + `gofmt`
clean; mobile `npm install` + `typecheck` exit 0; no Go tests yet; `mobile_dump.txt` deleted pre-existing
(left untouched). No fix commit needed.

**Phase 2 — Dashboard UI polish + folder browser (dashboard binary / web layer).**
Polish `agent-web/` in place: hierarchy, spacing, typography, cards + hover, buttons, scrollbars,
transfer rows, paired-client rows, empty/loading/error states, responsive, subtle transitions.
**Folder browser:** backend `agent/dashboard/browse.go` → loopback-only `GET /ui/fs/list` (dirs +
quick locations + parent nav) and `POST /ui/fs/mkdir`; frontend modal (open, list, quick locations,
parent nav, create folder, "Use this folder" fills input, Save persists via `POST /ui/settings/
output-dir`). Verify dashboard loads, QR, live progress, paired clients, cancel, unpair, narrow
responsive, LAN→`/ui/fs/list` returns 403. Commit: `feat(agent-web): polish dashboard and add folder browser`.

**Phase 3 — Unified `lanlink` binary + core/terminal decoupling.**
Extract sender `cli/send_file.go` → `internal/transfer/sender.go` (progress callback), keep
`cli/send_file.go` thin. Refactor receiver into `internal/agentserver` (move core handlers + the
transfer-state registry out of `agent/dashboard`; repoint `terminal_progress.go`); keep
`go build ./agent` via a thin `agent/main.go` (core + dashboard routes via injected hook). New
`cmd/lanlink/main.go` router (no agent-web). Audit/fix terminal bugs (R10). Verify no agent-web in
`go list -deps ./cmd/lanlink`; headless receive accepts CLI + mobile upload; send uploads;
desktop→desktop; `agent` still serves /ui; mobile unaffected. Commits: `refactor(agent): move transfer
state into pure-Go core (no agent-web)`, `refactor(transfer): extract reusable sender`,
`feat(cli): unified lanlink terminal binary`, + `fix(...)`.

**Phase 4 — LAN discovery & tokenless auto-connect.**
`internal/discovery/{beacon.go,scan.go}` (pure Go): `Announce(ctx,info)` per-interface every ~2s on
UDP `LANLINK_DISCOVERY_PORT` (default 8788); `Scan(timeout)` deduped hosts, graceful empty. Core:
Announce by default (`--no-discovery`); `POST /pair/auto` (LAN-facing) issues creds via
`internal/auth`+`internal/store`, recorded in core state; QR/token intact; additive `protocol/` DTOs.
Terminal `lanlink scan` lists + auto-connects (`--connect <name|addr>`). Dashboard "Scan network" →
loopback-only `GET /ui/discovery/scan`. Mobile `app/scan.tsx` + Scan entry on `pair.tsx`: HTTP
`/health` subnet sweep → auto-connect via `/pair/auto` → device screen. Commits: discovery package,
agent announce/open-pairing, CLI scan, dashboard scan UI, mobile scan.

**Phase 5 — Mobile polish (only where useful).** Status clarity, empty/error states, pairing + scan/QR
choice clarity, cancel/retry feedback, progress, responsiveness; `app.json` version/`android.package`;
`eas.json` APK profile. Preserve pairing/upload. Verify `npm run typecheck` + flows.
Commit: `feat(mobile): polish transfer/pairing screens and states`.

**Phase 6 — Release scripts & docs.** `scripts/build-release.sh` (CGO disabled): `release/
lanlink-linux-amd64`, `release/lanlink-windows-amd64.exe` (optional arm64), + optional dashboard
binary. `mobile/eas.json` APK profile. `docs/release.md` (Linux/Windows exe, Android APK/EAS, local
test checklist, files to attach, known limits). Verify outputs + `lanlink-linux-amd64 receive` runs.
Commit: `chore(release): add cross-platform build scripts and docs`.

**Phase 7 — Final verification.** `go fmt/vet/test ./...`; build `./agent` + `./cmd/lanlink`;
`go list -deps ./cmd/lanlink | grep -c agent-web` == 0; mobile `npm run typecheck`; discovery/sender
tests; `git status`; `git log --oneline`. Full manual checklist from `/goal` (document blockers).
Final report. No push.

## Files likely to change / be added
- Core split: new `internal/agentserver/*` (from `agent/*.go` + `agent/dashboard/state.go`); thin
  `agent/main.go`; trimmed `agent/dashboard/{routes,qr,browser,settings}.go`; new `agent/dashboard/browse.go`.
- Terminal: new `cmd/lanlink/main.go`; new `internal/transfer/sender.go`; edit `cli/{send_file.go,main.go}`.
- Dashboard UI: `agent-web/{index.html,assets/styles.css,assets/app.js}`.
- Discovery: new `internal/discovery/{beacon.go,scan.go}`; `POST /pair/auto`; new `protocol/pair_auto.go`;
  `GET /ui/discovery/scan`; `lanlink scan`; dashboard scan UI; mobile `app/scan.tsx` + `pair.tsx` + api
  helper; `LANLINK_DISCOVERY_PORT` in `internal/config` + `.env.example`.
- Mobile: `mobile/app/(tabs)/{transfers,device,settings}.tsx`, `app/pair.tsx`, `app.json`, `eas.json`.
- Release: `scripts/build-release.sh`, `docs/release.md`, README.

## Test plan
Go `vet`/`build`/`test ./...` each phase; first unit tests for sender, receiver manager, discovery;
the agent-web-free `go list -deps` check. Mobile `npm run typecheck`. Manual e2e: `lanlink receive`
(headless) + `agent` (dashboard), browse/scan/pair/transfer/cancel/unpair, CLI + mobile send,
tokenless auto-connect, loopback-only on `/ui/fs/*` and `/ui/discovery/scan`.

## Commit plan
~15–18 small, logical, verified commits. Never commit broken code; never mix unrelated changes; all
local, no push.
