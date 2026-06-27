# Changelog

All notable changes to LANLink are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project follows
[Semantic Versioning](https://semver.org/).

## [1.1.0] — 2026-06-28

### Mobile (Android)
- **Android share sheet:** LANLink now appears in the system share sheet. Share a
  file (or several) from any app — "Share → LANLink" — and it's sent to the
  currently paired receiver, or queued and sent the moment you pair. Implemented
  as a local native module (`modules/lanlink-share`) that streams the shared
  `content://` URIs directly (never copied), plus the `withShareIntent` config
  plugin that registers the intent filters.

### Desktop
- **Terminal binary ships as `lanlink` / `lanlink.exe`** (no `<os>-<arch>` suffix
  on amd64), so a downloaded binary runs directly as `./lanlink receive` /
  `./lanlink send` on Linux and Windows. arm64 terminal builds are suffixed
  `lanlink-arm64`. The web/dashboard build keeps its `lanlink-<os>-<arch>` name.

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
