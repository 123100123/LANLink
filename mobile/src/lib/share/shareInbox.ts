import { router } from "expo-router";

import { enqueueFiles } from "@/lib/transfer/transferManager";
import { useSessionStore } from "@/store/sessionStore";
import type { SharedFile } from "../../../modules/lanlink-share";

type QueuedFile = {
  uri: string;
  name: string;
  size: number;
  mimeType?: string;
};

// Files shared while the app isn't paired yet. Held until pairing completes,
// then flushed by flushPendingShares(). The system grants this process read
// access to the URIs for the lifetime of the task, so they stay readable across
// the pairing detour as long as the app stays open.
let pending: QueuedFile[] = [];

function toQueued(files: SharedFile[]): QueuedFile[] {
  return files
    .filter((f) => Boolean(f.uri))
    .map((f) => ({
      uri: f.uri,
      name: f.name || "shared file",
      size: f.size || 0,
      mimeType: f.mimeType ?? undefined,
    }));
}

// On a cold start from the share sheet this runs right after hydration, which
// can be before the root navigator has mounted — navigating then throws, and in
// a release build that uncaught startup error leaves a dead black screen. Retry
// a few frames instead of crashing; the files are already queued either way.
function safeReplace(href: "/(tabs)/transfers" | "/pair", attempt = 0): void {
  try {
    router.replace(href);
  } catch {
    if (attempt < 20) {
      setTimeout(() => safeReplace(href, attempt + 1), 50);
    }
  }
}

function sendNow(files: QueuedFile[]): boolean {
  const { credentials, agentAddress } = useSessionStore.getState();
  if (!credentials?.authToken || !agentAddress) return false;
  enqueueFiles(files, agentAddress, credentials.authToken);
  safeReplace("/(tabs)/transfers");
  return true;
}

/**
 * Handle files arriving from the system share sheet. Auto-sends to the currently
 * paired ("last") device; if no device is paired, stashes them and routes to the
 * pairing screen so they send the moment pairing succeeds.
 */
export function receiveSharedFiles(files: SharedFile[]): void {
  const queued = toQueued(files);
  if (queued.length === 0) return;

  if (!sendNow(queued)) {
    pending.push(...queued);
    safeReplace("/pair");
  }
}

/** Flush files that were shared before a device was paired. */
export function flushPendingShares(): void {
  if (pending.length === 0) return;
  const files = pending;
  if (sendNow(files)) {
    pending = [];
  }
}
