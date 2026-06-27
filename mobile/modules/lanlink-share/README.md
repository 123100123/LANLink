# lanlink-share

Local Expo native module (Android only) that lets LANLink appear in the Android
system **share sheet**. When the user shares a file from any app ("Share → LANLink"),
this module hands the shared `content://` URIs to JS, which queues them for upload
to the currently paired receiver.

It pairs with two other pieces:

- **`plugins/withShareIntent.js`** — a config plugin that registers the
  `ACTION_SEND` / `ACTION_SEND_MULTIPLE` intent filters (`*/*`) on `MainActivity`
  at prebuild, so Android offers LANLink as a share target.
- **`src/lib/share/`** — the JS glue that routes shared files to the transfer
  queue (auto-sending to the last paired device), or to the pairing screen when
  no device is paired yet.

## API

```ts
import {
  isAvailable,
  getInitialShareIntent,
  addShareListener,
  type SharedFile,
} from "../../modules/lanlink-share";

// Cold start: files the app was launched with from the share sheet.
const files = await getInitialShareIntent(); // SharedFile[]

// Warm start: app already open when the user shares.
const sub = addShareListener((files) => { /* ... */ });
sub.remove();
```

`SharedFile` is `{ uri, name, size, mimeType }`. The URIs are streamed straight to
the receiver by `lanlink-uploader` and are never copied into app cache.

## Notes

- Android only; `isAvailable` is `false` on iOS / web / Expo Go.
- Requires a dev/EAS build (not Expo Go) because it ships native code and a
  prebuild config plugin.
- `MainActivity` is `singleTask`, so a share into an already-running app arrives
  via `onNewIntent` (handled by `OnNewIntent` → `onShare` event); a share into a
  cold app arrives as the launch intent (handled by `getInitialShareIntent`).
