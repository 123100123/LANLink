import { requireOptionalNativeModule } from "expo-modules-core";

export type SharedFile = {
  /** content:// or file:// URI of the shared file. Streamed directly, never copied. */
  uri: string;
  /** Display name resolved from the content provider (falls back to "shared file"). */
  name: string;
  /** Size in bytes if the provider reported it, else 0. */
  size: number;
  /** MIME type reported by the content resolver, if any. */
  mimeType?: string | null;
};

type ShareSubscription = { remove(): void };

type LanlinkShareNativeModule = {
  getInitialShareIntent(): Promise<SharedFile[]>;
  addListener(
    event: "onShare",
    listener: (payload: { files: SharedFile[] }) => void
  ): ShareSubscription;
};

// Null on iOS / web / Expo Go, so the share wiring is a no-op there.
const nativeModule =
  requireOptionalNativeModule<LanlinkShareNativeModule>("LanlinkShare");

/** True when the native Android share-intent receiver is linked into this build. */
export const isAvailable: boolean = nativeModule != null;

/**
 * Returns the files the app was cold-launched with from the system share sheet,
 * or [] if it wasn't launched from a share. Marks the launch intent consumed so a
 * later remount doesn't re-deliver the same files.
 */
export function getInitialShareIntent(): Promise<SharedFile[]> {
  if (!nativeModule) return Promise.resolve([]);
  return nativeModule.getInitialShareIntent();
}

/**
 * Subscribes to files shared while the app is already running (warm start).
 * Returns a subscription; call remove() to stop listening. No-op when the native
 * module is unavailable.
 */
export function addShareListener(
  listener: (files: SharedFile[]) => void
): ShareSubscription {
  if (!nativeModule) return { remove() {} };
  return nativeModule.addListener("onShare", ({ files }) => listener(files));
}
