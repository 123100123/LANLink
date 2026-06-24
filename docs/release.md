# LANLink release guide

This guide covers building release artifacts for the desktop binaries (Linux &
Windows) and the Android app.

LANLink ships **two desktop binaries**, both built from the same Go source:

| Binary | Purpose | Web UI |
|--------|---------|--------|
| `lanlink` | Pure-Go terminal app: `receive`, `send`, `scan`, `pair`, … | none (zero `agent-web` dependency) |
| `lanlink-agent` | Receiver **with** the web dashboard at `/ui` | embeds `agent-web` |

Use `lanlink` for a headless / scriptable / minimal install, and `lanlink-agent`
when you want the browser dashboard.

## Prerequisites

- Go 1.24+ (`go version`)
- For the Android app: Node 18+, an [Expo](https://expo.dev) account, and the
  EAS CLI (`npm install -g eas-cli`)

## Desktop builds (Linux & Windows)

All desktop builds are **pure Go** (`CGO_ENABLED=0`) and cross-compile without a
toolchain for the target OS.

### One command (recommended)

```bash
scripts/build-release.sh
```

This writes to `./release/`:

```
lanlink-linux-amd64
lanlink-windows-amd64.exe
lanlink-agent-linux-amd64
lanlink-agent-windows-amd64.exe
```

Options:

- `LANLINK_ARM64=1 scripts/build-release.sh` — also build `linux/arm64` and `windows/arm64`.
- `LANLINK_VERSION=0.5.0 scripts/build-release.sh` — stamp a version string in the log.

### Manual builds

Linux executable:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o release/lanlink-linux-amd64 ./cmd/lanlink
```

Windows executable:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o release/lanlink-windows-amd64.exe ./cmd/lanlink
```

Swap `./cmd/lanlink` for `./agent` to build the dashboard variant.

### App icon (Windows)

The Windows `.exe`s embed the LANLink icon via committed `.syso` resource files
(`cmd/lanlink/icon_windows_amd64.syso`, `agent/icon_windows_*.syso`). Linux ELF
binaries cannot carry an icon. To regenerate the resources after changing the
icon (`assets/icon.ico`):

```bash
go install github.com/akavel/rsrc@latest
for d in cmd/lanlink agent; do for a in amd64 arm64; do
  "$(go env GOPATH)/bin/rsrc" -ico assets/icon.ico -arch $a -o "$d/icon_windows_$a.syso"
done; done
```

## Android app (APK)

The mobile app is a managed Expo app. It stays Expo-Go / managed-workflow safe
(no custom native modules), so an APK can be produced with EAS Build or a local
Gradle build. The app icon, splash, package id (`com.lanlink.app`) and version
are already configured in `mobile/app.json`.

The mobile app is a managed Expo app. It stays Expo-Go / managed-workflow safe
(no custom native modules), so an APK can be produced with EAS Build.

### EAS cloud build (recommended)

```bash
cd mobile
npm install
eas login
eas build --profile preview --platform android
```

The `preview` profile (in `mobile/eas.json`) produces an installable **APK**
(`buildType: apk`, internal distribution). When the build finishes, EAS prints a
download URL for the `.apk`.

The `production` profile produces an **AAB** (app bundle) for Play Store
submission instead.

### Local APK build (Android SDK + Java 17)

```bash
cd mobile
npx expo prebuild -p android --no-install      # generates android/
cd android
./gradlew assembleDebug                          # → app/build/outputs/apk/debug/app-debug.apk
```

Copy the result to `release/lanlink-<version>-debug.apk`. The debug APK is signed
with the auto-generated debug keystore and is sideloadable (`adb install`).

> **Note:** the local Gradle build must reach Google's Maven repo
> (`dl.google.com/dl/android/maven2`) to fetch the Android Gradle Plugin. Behind
> some VPNs/proxies that host returns errors — disable the VPN or use the EAS
> cloud build above if `./gradlew` fails with "Could not find
> com.android.tools.build:gradle".

### App configuration

Relevant fields in `mobile/app.json`:

- `version` — user-facing version (`0.5.0`)
- `android.package` — `com.lanlink.app`
- `android.versionCode` — integer, bump for each store submission
- `ios.bundleIdentifier` — `com.lanlink.app`

The app icon, adaptive icon, splash, and favicon are the LANLink WiFi mark in
`mobile/assets/` (generated from `assets/`); `app.json` already references them.

## Local release test checklist

After building, sanity-check the artifacts:

1. `./release/lanlink-linux-amd64 receive` — starts a receiver, prints a pairing
   token + QR in the terminal.
2. In another terminal: `./release/lanlink-linux-amd64 scan` — discovers the
   receiver and auto-connects (tokenless).
3. `./release/lanlink-linux-amd64 send <host:port> <file>` — uploads a file;
   confirm it lands in the output folder.
4. `./release/lanlink-agent-linux-amd64` — open `http://127.0.0.1:8787/ui`;
   confirm the dashboard loads, QR renders, folder browser works, and a transfer
   shows live progress.
5. Install the APK on a phone on the same Wi-Fi; pair via QR **or** "Scan
   network", then send a file.

## Files to attach to a release

- `lanlink-linux-amd64`
- `lanlink-windows-amd64.exe`
- `lanlink-agent-linux-amd64`
- `lanlink-agent-windows-amd64.exe`
- `lanlink-<version>-android.apk` (from EAS)
- (optional) `lanlink-linux-arm64`, `lanlink-windows-arm64.exe`

## Known limitations

- **Tokenless auto-connect** (`/pair/auto`, used by `scan`) is intentionally open
  on the local network. Run receivers only on trusted LANs, or start them with
  `lanlink receive --no-discovery` to disable beacon advertising. See the
  security note in the README.
- **Mobile discovery** uses an HTTP `/health` subnet sweep (`/24`), because
  managed Expo cannot listen for the UDP beacon without a native module. Desktop
  discovery uses the UDP beacon.
- Desktop dashboard filesystem/scan routes (`/ui/fs/*`, `/ui/discovery/scan`) are
  **loopback-only** and are never exposed to LAN clients.
