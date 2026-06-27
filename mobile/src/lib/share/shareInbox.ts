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

function sendNow(files: QueuedFile[]): boolean {
  const { credentials, agentAddress } = useSessionStore.getState();
  if (!credentials?.authToken || !agentAddress) return false;
  enqueueFiles(files, agentAddress, credentials.authToken);
  router.replace("/(tabs)/transfers");
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
    router.replace("/pair");
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
