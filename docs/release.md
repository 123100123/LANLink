# LANLink release guide

This guide covers building release artifacts for the desktop binaries (Linux &
Windows) and the Android app.

LANLink ships **two desktop binaries**, both built from the same Go source:

| Binary | Purpose | Web UI |
|--------|---------|--------|
| `lanlink-<os>-<arch>` | Runs a receiver **and** serves the dashboard, opening the browser on start | embeds `agent-web` (Windows `.exe` carries the app icon) |
| `lanlink-cmd-<os>-<arch>` | Pure-Go terminal app: `receive`, `send`, `pair`, … | none (zero `agent-web` dependency) |

Use `lanlink-…` when you want the browser dashboard, and `lanlink-cmd-…` for a
headless / scriptable / minimal terminal install.

## Prerequisites

- Go 1.24+ (`go version`)
- For the Android app: Node 18+ and an [Expo](https://expo.dev) account (the
  `eas-cli` is installed on demand via `npx eas-cli`).

## Desktop builds (Linux & Windows)

All desktop builds are **pure Go** (`CGO_ENABLED=0`) and cross-compile without a
toolchain for the target OS.

### One command (recommended)

```bash
scripts/build-release.sh
```

This writes to `./release/`:

```
lanlink-linux-amd64            # web build (opens the dashboard)
lanlink-windows-amd64.exe      # web build (opens the dashboard)
lanlink-cmd-linux-amd64        # terminal build
lanlink-cmd-windows-amd64.exe  # terminal build
```

Options:

- `LANLINK_ARM64=1 scripts/build-release.sh` — also build `linux/arm64` and `windows/arm64`.
- `LANLINK_VERSION=0.5.0 scripts/build-release.sh` — stamp a version string in the log.

### Manual builds

Web build (dashboard) — Linux and Windows:

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o release/lanlink-linux-amd64 ./agent
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o release/lanlink-windows-amd64.exe ./agent
```

Terminal build — swap `./agent` for `./cmd/lanlink` and name the output `lanlink-cmd-…`.

### App icon (Windows)

The web build's Windows `.exe` embeds the LANLink icon via committed `.syso`
resource files (`agent/icon_windows_amd64.syso`, `agent/icon_windows_arm64.syso`).
The terminal build carries no icon, and Linux ELF binaries cannot hold one. To
regenerate the resources after changing the icon (`assets/icon.ico`):

```bash
go install github.com/akavel/rsrc@latest
for a in amd64 arm64; do
  "$(go env GOPATH)/bin/rsrc" -ico assets/icon.ico -arch $a -o "agent/icon_windows_$a.syso"
done
```

## Android app (APK)

The mobile app is a managed Expo app (Expo-Go / managed-workflow safe, no custom
native modules). The icon, splash, package id (`com.lanlink.app`), version, and
EAS `projectId` are configured in `mobile/app.json`.

### EAS cloud build (recommended)

```bash
cd mobile
npx eas-cli build --profile preview --platform android
```

The `preview` profile (in `mobile/eas.json`) produces an installable **APK**
(`buildType: apk`). EAS builds on Expo's infrastructure and prints a download URL
for the `.apk` when finished; save it as `release/lanlink.apk`. The `production`
profile produces an **AAB** for Play Store submission instead.

### Local APK build (Android SDK + Java 17)

```bash
cd mobile
npx expo prebuild -p android --no-install      # generates android/
cd android
./gradlew assembleDebug                          # → app/build/outputs/apk/debug/app-debug.apk
```

> **Note:** the local Gradle build must reach Google's Maven repo
> (`dl.google.com/dl/android/maven2`) to fetch the Android Gradle Plugin. Behind
> some VPNs/proxies that host is unreachable — disable the VPN or use the EAS
> cloud build above if `./gradlew` fails with "Could not find
> com.android.tools.build:gradle".

## Local release test checklist

After building, sanity-check the artifacts:

1. `./release/lanlink-cmd-linux-amd64 receive` — starts a receiver, prints a
   pairing token + QR in the terminal.
2. `./release/lanlink-cmd-linux-amd64 pair <host:port> <token>` then
   `send <host:port> <file>` — uploads a file; confirm it lands in the output folder.
3. `./release/lanlink-linux-amd64` — opens `http://127.0.0.1:8787/ui` in the
   browser; confirm the dashboard loads, QR renders, the folder browser works,
   and a transfer shows live progress.
4. Install the APK on a phone on the same Wi-Fi; pair by scanning the agent QR
   (or entering the address + token), then send a file.

## Files to attach to a release

- `lanlink-linux-amd64`, `lanlink-windows-amd64.exe` (web build)
- `lanlink-cmd-linux-amd64`, `lanlink-cmd-windows-amd64.exe` (terminal build)
- `lanlink.apk` (from EAS)
- (optional) `lanlink-linux-arm64`, `lanlink-windows-arm64.exe`, plus their `lanlink-cmd-…` variants

## Known limitations

- Pairing is by QR or manual address + token (no LAN auto-discovery in this build).
- Desktop dashboard filesystem routes (`/ui/fs/*`) are **loopback-only** and are
  never exposed to LAN clients.
