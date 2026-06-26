import { requireOptionalNativeModule } from "expo-modules-core";

export type UploadFileOptions = {
  /** Full URL of the receiver upload endpoint, e.g. http://10.0.0.5:8787/transfers/upload */
  url: string;
  /** Source URI of the file to upload (file:// or content://). Never copied to cache. */
  uri: string;
  filename: string;
  transferId: string;
  authToken: string;
  /** File size in bytes if known; sent as X-File-Size and used for Content-Length. */
  size?: number;
  mimeType?: string;
};

export type UploadFileResult = {
  /** HTTP status code returned by the receiver. */
  status: number;
  /** Raw response body (the receiver returns JSON). */
  body: string;
};

type LanlinkUploaderNativeModule = {
  uploadFile(options: UploadFileOptions): Promise<UploadFileResult>;
  cancelUpload(transferId: string): Promise<boolean>;
};

// requireOptionalNativeModule returns null when the native module isn't present
// (iOS, web, or Expo Go), so callers can fall back to a JS uploader.
const nativeModule = requireOptionalNativeModule<LanlinkUploaderNativeModule>(
  "LanlinkUploader"
);

/** True when the native Android streaming uploader is linked into this build. */
export const isAvailable: boolean = nativeModule != null;

/**
 * Streams a file straight to the receiver from a native OkHttp client, bypassing
 * React Native's networking module (whose ProgressRequestBody writes uploads one
 * byte at a time). Android only; throws if the native module is unavailable.
 */
export function uploadFile(
  options: UploadFileOptions
): Promise<UploadFileResult> {
  if (!nativeModule) {
    throw new Error(
      "LanlinkUploader native module is not available on this platform/build"
    );
  }
  return nativeModule.uploadFile(options);
}

/** Cancels an in-flight upload by transferId. No-op (resolves false) if unavailable. */
export function cancelUpload(transferId: string): Promise<boolean> {
  if (!nativeModule) return Promise.resolve(false);
  return nativeModule.cancelUpload(transferId);
}
