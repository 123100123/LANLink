# lanlink-uploader (local Expo native module)

Native Android streaming uploader for lanlink. It POSTs a picked file straight to
the receiver's `/transfers/upload` endpoint using its **own OkHttp client and a
custom streaming `RequestBody`** (1 MiB buffer), reading the source URI via
`ContentResolver.openInputStream`.

## Why this exists

React Native's Android `NetworkingModule` wraps **every** upload body in
`ProgressRequestBody`, whose `FilterOutputStream` writes the body **one byte at a
time** with a progress check per byte. That throttles `fetch`/`XMLHttpRequest`/
`FormData` and even some library uploads to ~1/7th of link speed (measured 6 MB/s
vs ~25–42 MB/s for native `curl` on the same device/link). This module bypasses
RN networking entirely, so uploads run at link speed.

## API

```ts
// Imported by relative path from app code (see transferManager.ts):
//   import * as NativeUploader from "../../../modules/lanlink-uploader";
import { uploadFile, cancelUpload, isAvailable } from "./modules/lanlink-uploader";

const { status, body } = await uploadFile({
  url, uri, filename, transferId, authToken, size, mimeType,
});
await cancelUpload(transferId);
```

- Streams the URI directly — **never copies the file into app cache** (large
  `content://` files do not crash).
- Sends headers: `Authorization: Bearer <token>`, `X-Filename`, `X-Transfer-Id`,
  `X-File-Size` (when size is known), `Content-Type: application/octet-stream`.
- Returns the receiver's HTTP `status` and raw `body`.
- `cancelUpload(transferId)` cancels the in-flight OkHttp call.

## Build / runtime notes

- **Android only.** `isAvailable` is `false` on iOS/web and in **Expo Go**, which
  cannot load custom native modules. Callers must fall back (the app falls back to
  `expo-file-system` — see `mobile/src/lib/transfer/transferManager.ts`).
- Requires a **dev build** or an **EAS build** (`eas build -p android --profile
  preview`). It is autolinked via `expoAutolinking.useExpoModules()`; after adding
  it, rebuild the Android app (a JS-only reload will not pick up the native code).
- The receiver already accepts a raw request body, so **no server change** is
  required.
