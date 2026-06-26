import * as FileSystem from "expo-file-system/legacy";
import { Platform } from "react-native";
import { httpUrl } from "@/lib/api/endpoints";
import { createId } from "@/lib/protocol/envelope";
import { useTransferStore, type TransferItem } from "@/store/transferStore";
import * as NativeUploader from "../../../modules/lanlink-uploader";

// How often the phone asks the receiver how many bytes it has actually written.
const SERVER_POLL_INTERVAL_MS = 400;

let activeTask: FileSystem.UploadTask | null = null;
let activeNativeId: string | null = null;
let queueLoopRunning = false;
let intentionallyCancelled = false;

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// The receiver's authoritative view of a transfer: how many bytes it has truly
// written to disk and the throughput it measured. The phone displays these so
// its speed/bytes/ETA match the desktop instead of its own send-buffer count.
async function fetchServerProgress(
  item: TransferItem
): Promise<{ received: number; total: number; speed: number; state: string } | null> {
  try {
    const url = httpUrl(
      item.agentAddress,
      `/transfers/${encodeURIComponent(item.id)}/status`
    );
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${item.authToken}` },
    });
    if (!res.ok) return null;
    const j = (await res.json()) as {
      received?: number;
      total?: number;
      speed?: number;
      state?: string;
    };
    return {
      received: j.received ?? 0,
      total: j.total ?? 0,
      speed: j.speed ?? 0,
      state: j.state ?? "",
    };
  } catch {
    return null;
  }
}

function getNextWaiting(): TransferItem | undefined {
  return useTransferStore.getState().transfers.find(
    (t) => t.status === "waiting"
  );
}

function getActiveUpload(): TransferItem | undefined {
  return useTransferStore.getState().transfers.find(
    (t) => t.status === "uploading"
  );
}

function abortActive(): void {
  if (activeNativeId) {
    try {
      NativeUploader.cancelUpload(activeNativeId);
    } catch {}
    activeNativeId = null;
  }
  if (activeTask) {
    try {
      activeTask.cancelAsync();
    } catch {}
    activeTask = null;
  }
}

export function enqueueFiles(
  files: { uri: string; name: string; size: number; mimeType?: string }[],
  agentAddress: string,
  authToken: string
): number {
  const items: TransferItem[] = files.map((f) => ({
    id: createId("transfer"),
    uri: f.uri,
    filename: f.name,
    mimeType: f.mimeType,
    size: f.size,
    agentAddress,
    authToken,
    sentBytes: 0,
    progress: 0,
    status: "waiting" as const,
    speed: 0,
    elapsed: 0,
    startedAt: Date.now(),
  }));

  useTransferStore.getState().addTransfers(items);
  processQueue();
  return items.length;
}

export function cancelTransfer(id: string): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  if (!t) return;

  const active = getActiveUpload();
  if (active && active.id === id) {
    intentionallyCancelled = true;
    abortActive();
  }
  useTransferStore.getState().updateTransfer(id, {
    status: "cancelled",
    completedAt: Date.now(),
  });
}

export function retryTransfer(id: string): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  if (!t || (t.status !== "failed" && t.status !== "cancelled")) return;
  useTransferStore.getState().updateTransfer(id, {
    status: "waiting",
    error: undefined,
    sentBytes: 0,
    progress: 0,
    speed: 0,
    elapsed: 0,
    completedAt: undefined,
  });
  processQueue();
}

export function removeTransfer(id: string): void {
  const active = getActiveUpload();
  if (active && active.id === id) {
    intentionallyCancelled = true;
    abortActive();
  }
  useTransferStore.getState().removeTransfer(id);
}

export function stopAll(): void {
  const store = useTransferStore.getState();
  const active = getActiveUpload();

  if (active) {
    intentionallyCancelled = true;
    abortActive();
    store.updateTransfer(active.id, {
      status: "cancelled",
      completedAt: Date.now(),
    });
  }

  for (const t of store.transfers) {
    if (t.status === "waiting") {
      store.updateTransfer(t.id, {
        status: "cancelled",
        completedAt: Date.now(),
      });
    }
  }
}

export function startAll(): void {
  const store = useTransferStore.getState();
  let hasRequeued = false;

  for (const t of store.transfers) {
    if (t.status === "cancelled" || t.status === "failed") {
      store.updateTransfer(t.id, {
        status: "waiting",
        error: undefined,
        sentBytes: 0,
        progress: 0,
        speed: 0,
        elapsed: 0,
        completedAt: undefined,
      });
      hasRequeued = true;
    }
  }

  if (hasRequeued) {
    processQueue();
  }
}

export function clearCompleted(): void {
  useTransferStore.getState().clearCompleted();
}

function processQueue(): void {
  if (queueLoopRunning) return;
  if (getActiveUpload()) return;

  queueLoopRunning = true;

  (async () => {
    try {
      while (true) {
        const next = getNextWaiting();
        if (!next) break;
        useTransferStore
          .getState()
          .updateTransfer(next.id, { status: "uploading" });
        await runStreamingUpload(next);
      }
    } finally {
      queueLoopRunning = false;
    }
  })();
}

// Whether to use the native Android streaming uploader. It's only present in dev/
// EAS builds (not Expo Go / iOS / web), so callers fall back to expo-file-system.
function useNativeUploader(): boolean {
  return Platform.OS === "android" && NativeUploader.isAvailable;
}

// PRIMARY (Android): stream the file straight to the receiver from a native OkHttp
// client (the local lanlink-uploader module). This bypasses React Native's
// NetworkingModule, whose ProgressRequestBody wraps EVERY upload body in a
// FilterOutputStream that writes ONE BYTE AT A TIME with a per-byte progress check
// — throttling fetch/XHR/FormData AND expo-file-system uploads to ~1/7th of link
// speed (measured 6 MB/s vs ~25-42 MB/s for native curl on the same link). The
// native module streams the URI directly (no cache copy) at link speed.
async function uploadViaNative(item: TransferItem): Promise<string> {
  activeNativeId = item.id;
  try {
    const result = await NativeUploader.uploadFile({
      url: httpUrl(item.agentAddress, "/transfers/upload"),
      uri: item.uri,
      filename: item.filename,
      transferId: item.id,
      authToken: item.authToken,
      size: item.size > 0 ? item.size : undefined,
      mimeType: item.mimeType,
    });
    activeNativeId = null;

    if (result.status < 200 || result.status >= 300) {
      let msg = `Upload failed (${result.status})`;
      try {
        const j = JSON.parse(result.body) as { error?: string };
        if (j.error) msg = j.error;
      } catch {}
      throw new Error(msg);
    }
    return result.body;
  } catch (err) {
    activeNativeId = null;
    throw err;
  }
}

// FALLBACK (iOS / Expo Go / native module unavailable): upload via expo-file-system's
// own OkHttp client (BINARY_CONTENT raw body). Also avoids RN's per-byte path, but
// the native module above is preferred on Android. The Go receiver accepts a raw
// body either way, so no server change is needed. Progress is polled from the
// receiver in runStreamingUpload, so we pass no progress callback here.
async function uploadViaFileSystem(item: TransferItem): Promise<string> {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${item.authToken}`,
    "X-Filename": item.filename,
    "X-Transfer-Id": item.id,
  };
  if (item.size > 0) {
    headers["X-File-Size"] = String(item.size);
  }

  const task = FileSystem.createUploadTask(
    httpUrl(item.agentAddress, "/transfers/upload"),
    item.uri,
    {
      httpMethod: "POST",
      uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
      headers,
    }
  );
  activeTask = task;

  const result = await task.uploadAsync();
  activeTask = null;

  // A cancelled task resolves without a result.
  if (!result) {
    throw new Error("cancelled");
  }

  if (result.status < 200 || result.status >= 300) {
    let msg = `Upload failed (${result.status})`;
    try {
      const j = JSON.parse(result.body) as { error?: string };
      if (j.error) msg = j.error;
    } catch {}
    throw new Error(msg);
  }

  return result.body;
}

// Picks the fastest available uploader for this platform/build.
function performUpload(item: TransferItem): Promise<string> {
  return useNativeUploader() ? uploadViaNative(item) : uploadViaFileSystem(item);
}

async function runStreamingUpload(item: TransferItem): Promise<void> {
  const startTime = Date.now();
  intentionallyCancelled = false;

  // Benchmark logging: compare the native uploader vs expo-file-system on the same
  // link (URI scheme, size, duration, avg MB/s, response status are logged below).
  const uriScheme = item.uri.split(":")[0] || "unknown";
  const mechanism = useNativeUploader() ? "native" : "expo-fs";
  console.log(
    `[upload] start id=${item.id} file="${item.filename}" scheme=${uriScheme} ` +
      `size=${item.size} via=${mechanism}`
  );

  // The phone can't measure real upload progress itself (and adding a native
  // progress callback re-introduces per-chunk request-body wrapping). So the
  // receiver is the source of truth — poll it for the real received-byte count and
  // throughput, and display those. This makes the phone's speed / bytes / ETA
  // match the desktop exactly.
  let polling = true;

  const pollLoop = async () => {
    while (polling) {
      await delay(SERVER_POLL_INTERVAL_MS);
      if (!polling) break;
      const prog = await fetchServerProgress(item);
      if (!polling || !prog) continue;
      if (prog.received > 0 || prog.total > 0) {
        const total = prog.total > 0 ? prog.total : item.size;
        useTransferStore.getState().updateTransfer(item.id, {
          sentBytes: prog.received,
          size: total > 0 ? total : item.size,
          // Hold just under 100% mid-transfer; completion is set on the server's
          // confirmed save in the onload branch below.
          progress: total > 0 ? Math.min(0.99, prog.received / total) : 0,
          speed: prog.speed,
          elapsed: (Date.now() - startTime) / 1000,
        });
      }
    }
  };
  void pollLoop();

  try {
    const body = await performUpload(item);
    polling = false;

    if (intentionallyCancelled) return;

    const json = JSON.parse(body) as {
      status: string;
      path?: string;
      received?: number;
      speed?: number;
      error?: string;
    };

    if (json.status !== "saved") {
      throw new Error(json.error || "Upload failed");
    }

    // Final numbers come straight from the receiver: the bytes it actually wrote
    // and the throughput it measured at the last byte. Falling back to local
    // wall-time math only if the server omitted them.
    const endElapsed = (Date.now() - startTime) / 1000;
    const finalReceived =
      typeof json.received === "number" && json.received > 0
        ? json.received
        : item.size;
    const finalSpeed =
      typeof json.speed === "number" && json.speed > 0
        ? json.speed
        : endElapsed > 0
          ? finalReceived / endElapsed
          : 0;
    console.log(
      `[upload] done id=${item.id} via=${mechanism} bytes=${finalReceived} ` +
        `dur=${endElapsed.toFixed(2)}s avg=${(
          finalReceived /
          1e6 /
          Math.max(endElapsed, 0.001)
        ).toFixed(1)}MB/s status=${json.status}`
    );
    useTransferStore.getState().updateTransfer(item.id, {
      status: "completed",
      progress: 1,
      sentBytes: finalReceived,
      size: item.size > 0 ? item.size : finalReceived,
      speed: finalSpeed,
      elapsed: endElapsed,
      completedAt: Date.now(),
      savedPath: json.path,
    });
  } catch (err) {
    polling = false;
    if (intentionallyCancelled) return;

    const msg = err instanceof Error ? err.message : "Upload failed";
    useTransferStore.getState().updateTransfer(item.id, {
      status: "failed",
      error: msg,
      completedAt: Date.now(),
    });
  } finally {
    polling = false;
  }
}
