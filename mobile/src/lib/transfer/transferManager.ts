import { httpUrl } from "@/lib/api/endpoints";
import { createId } from "@/lib/protocol/envelope";
import { useTransferStore, type TransferItem } from "@/store/transferStore";

const PROGRESS_UPDATE_INTERVAL_MS = 250;
const PROGRESS_UPDATE_MIN_DELTA = 0.01;

let activeXhr: XMLHttpRequest | null = null;
let queueLoopRunning = false;
let intentionallyCancelled = false;

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
  if (activeXhr) {
    try {
      activeXhr.abort();
    } catch {}
    activeXhr = null;
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

// Streams the picked file straight to the receiver via FormData. React Native's
// native networking opens the (content:// or file://) URI and streams it through
// okhttp without buffering the whole file into memory — so large files upload
// fast and never freeze the UI on a cache copy.
function uploadViaXhr(
  item: TransferItem,
  onProgress: (sentBytes: number) => void
): Promise<string> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    activeXhr = xhr;

    xhr.open("POST", httpUrl(item.agentAddress, "/transfers/upload"));
    xhr.setRequestHeader("Authorization", `Bearer ${item.authToken}`);
    xhr.setRequestHeader("X-Filename", item.filename);
    xhr.setRequestHeader("X-Transfer-Id", item.id);
    if (item.size > 0) {
      // content:// URIs often report no content-length, so tell the server the
      // real size for an accurate progress total.
      xhr.setRequestHeader("X-File-Size", String(item.size));
    }

    xhr.upload.onprogress = (e) => {
      // e.total is unreliable for content:// (can be 0); progress is computed
      // against item.size in the caller.
      onProgress(e.loaded);
    };

    xhr.onload = () => {
      activeXhr = null;
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(xhr.responseText);
        return;
      }
      let msg = `Upload failed (${xhr.status})`;
      try {
        const j = JSON.parse(xhr.responseText) as { error?: string };
        if (j.error) msg = j.error;
      } catch {}
      reject(new Error(msg));
    };

    xhr.onerror = () => {
      activeXhr = null;
      reject(new Error("Network error"));
    };

    xhr.onabort = () => {
      activeXhr = null;
      reject(new Error("cancelled"));
    };

    const form = new FormData();
    form.append("file", {
      uri: item.uri,
      name: item.filename,
      type: item.mimeType || "application/octet-stream",
    } as unknown as Blob);

    xhr.send(form);
  });
}

async function runStreamingUpload(item: TransferItem): Promise<void> {
  const startTime = Date.now();
  intentionallyCancelled = false;

  let lastUpdateTime = 0;
  let lastProgress = 0;

  const onProgress = (sentBytes: number) => {
    const now = Date.now();
    const total = item.size > 0 ? item.size : sentBytes;
    const p = total > 0 ? Math.min(1, sentBytes / total) : 0;

    if (
      now - lastUpdateTime >= PROGRESS_UPDATE_INTERVAL_MS ||
      Math.abs(p - lastProgress) >= PROGRESS_UPDATE_MIN_DELTA
    ) {
      const elapsed = (now - startTime) / 1000;
      useTransferStore.getState().updateTransfer(item.id, {
        sentBytes,
        size: total,
        progress: p,
        speed: elapsed > 0 ? sentBytes / elapsed : 0,
        elapsed,
      });
      lastUpdateTime = now;
      lastProgress = p;
    }
  };

  try {
    const body = await uploadViaXhr(item, onProgress);

    if (intentionallyCancelled) return;

    const json = JSON.parse(body) as {
      status: string;
      path?: string;
      received?: number;
      error?: string;
    };

    if (json.status !== "saved") {
      throw new Error(json.error || "Upload failed");
    }

    useTransferStore.getState().updateTransfer(item.id, {
      status: "completed",
      progress: 1,
      sentBytes: item.size,
      speed: 0,
      elapsed: (Date.now() - startTime) / 1000,
      completedAt: Date.now(),
      savedPath: json.path,
    });
  } catch (err) {
    if (intentionallyCancelled) return;

    const msg = err instanceof Error ? err.message : "Upload failed";
    useTransferStore.getState().updateTransfer(item.id, {
      status: "failed",
      error: msg,
      completedAt: Date.now(),
    });
  }
}
