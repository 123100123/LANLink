# Changelog

All notable changes to LANLink are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project follows
[Semantic Versioning](https://semver.org/).

## [1.1.1] — 2026-07-02

### Mobile (Android)
- **Fixed a black screen on reopening the app.** From the second launch onward
  the app could open to a blank screen. When Android recreates the activity
  while the JS process is still alive, expo-router 6 (SDK 54) failed to restore
  navigation state, leaving an empty navigator over the dark background.
  Backported the upstream fix ([expo/expo#42644]) via `patch-package`, and
  hardened the share-intent cold start so a queued share can't crash startup or
  be replayed from the recents list.
- **Fixed the receiver online/offline indicator.** The reachability check could
  hang indefinitely against an unreachable host, leaving the badge stuck on
  "Checking…". It now times out after 4 seconds and the Send screen re-probes
  the receiver periodically, so the status reflects reality.
- **Fixed transfer screen styling.** The live upload percentage was rendered in
  black (invisible on the dark card) and the action buttons used inconsistent
  colors. The percentage is now legible and every button follows the shared
  blue / red / navy palette used across the app.

### Tooling
- Consolidated on a single lockfile (`yarn.lock`) and removed the stale
  `package-lock.json`; native patches are applied on install via `patch-package`.

[expo/expo#42644]: https://github.com/expo/expo/pull/42644

## [1.1.0] — 2026-06-28

### Mobile (Android)
- **Android share sheet:** LANLink now appears in the system share sheet. Share a
  file (or several) from any app — "Share → LANLink" — and it's sent to the
  currently paired receiver, or queued and sent the moment you pair. Implemented
  as a local native module (`modules/lanlink-share`) that streams the shared
  `content://` URIs directly (never copied), plus the `withShareIntent` config
  plugin that registers the intent filters.

### Desktop
- **Two clearly named builds: `lanlink` (cmd) and `lanlinkAgent` (receiver UI).**
  - The terminal/cmd binary ships as `lanlink` / `lanlink.exe` (no `<os>-<arch>`
    suffix on amd64), so a downloaded binary runs directly as `./lanlink receive`
    / `./lanlink send` on Linux and Windows. arm64 cmd builds are suffixed
    `lanlink-arm64`.
  - The receiver-UI build (receiver + browser dashboard at `/ui`) is now named
    `lanlinkAgent-<os>-<arch>` (was `lanlink-<os>-<arch>`).

## [1.0.0] — 2026-06-26

First stable release.

### Desktop
- Unified pure-Go terminal binary `lanlink` — `receive` and `send`, with zero
  dependency on the web UI.
- Web dashboard build (embeds `agent-web`): live transfers, paired clients, and
  an output-folder browser, served at `/ui`.
- Pairing via the receiver's QR code or a manual address + token, with rotating
  tokens.
- Cross-compiles to Linux and Windows (amd64/arm64) with no cgo; the Windows
  executables carry the app icon.

### Mobile (Android / iOS, Expo)
- QR-code or address+token pairing, a file upload queue, and live
  progress/speed/ETA that mirror the desktop receiver.
- **Native Android streaming uploader** (`modules/lanlink-uploader`): streams the
  picked file straight to the receiver from its own OkHttp client, bypassing React
  Native's networking layer to upload at link speed. Falls back to
  `expo-file-system` where the native module is unavailable.

### Transfer
- HTTP data plane with raw-body and resumable/chunked uploads.
- New `GET /transfers/{id}/status` endpoint so senders display the receiver's
  authoritative received-byte count and throughput.

[1.1.0]: https://github.com/123100123/LANLink/releases/tag/v1.1.0
[1.0.0]: https://github.com/123100123/LANLink/releases/tag/v1.0.0
