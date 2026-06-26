# LANLink Mobile

The LANLink mobile client — an [Expo](https://expo.dev/) / React Native app for
Android and iOS. Pair with a desktop receiver by QR code or address + token,
then send files over the LAN with live progress.

## Requirements

- Node.js 18+ and Yarn (or npm)
- A running LANLink receiver on the same network (`lanlink receive`)
- For native builds: an [EAS](https://docs.expo.dev/eas/) account

## Develop

```bash
cd mobile
yarn install
yarn start            # Expo dev server
yarn typecheck        # tsc --noEmit
```

Open the project in a development build or Expo Go. Note: the native uploader
below requires a **development/EAS build** — Expo Go cannot load custom native
modules and the app transparently falls back to `expo-file-system`.

## Build (APK / app bundle)

```bash
eas build -p android --profile preview      # internal APK
eas build -p android --profile production   # store app bundle
```

## Native uploader

`modules/lanlink-uploader/` is a local Expo native module that streams uploads
from a native Android OkHttp client, bypassing React Native's networking layer
(which throttles uploads). See its [README](modules/lanlink-uploader/README.md).

## Layout

- `app/` — screens and navigation (Expo Router)
- `src/lib/` — transfer manager, API/socket clients, protocol, storage
- `src/store/` — Zustand state (session, transfers)
- `modules/lanlink-uploader/` — native Android streaming uploader
