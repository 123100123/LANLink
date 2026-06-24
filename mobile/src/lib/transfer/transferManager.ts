import * as FileSystem from "expo-file-system/legacy";
import { httpUrl } from "@/lib/api/endpoints";
import { createId } from "@/lib/protocol/envelope";
import { useTransferStore, type TransferItem } from "@/store/transferStore";

const PROGRESS_UPDATE_INTERVAL_MS = 250;
const PROGRESS_UPDATE_MIN_DELTA = 0.01;

let activeTask: FileSystem.UploadTask | null = null;
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

function isContentUri(uri: string): boolean {
  return uri.startsWith("content://");
}

async function getFreeDiskSpace(): Promise<number> {
  try {
    return await FileSystem.getFreeDiskStorageAsync();
  } catch {
    return -1;
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 * 1024 * 1024)
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

async function copyToCache(
  item: TransferItem
): Promise<{ uploadUri: string; isTemp: boolean }> {
  const freeSpace = await getFreeDiskSpace();
  if (
    freeSpace >= 0 &&
    item.size > 0 &&
    freeSpace < item.size + 1024 * 1024
  ) {
    throw new Error(
      `Not enough disk space. Need ${formatSize(item.size)}, only ${formatSize(freeSpace)} available.`
    );
  }

  const cacheDir = FileSystem.cacheDirectory ?? "";
  const tempPath = `${cacheDir}upload_${item.id}_${item.filename}`;
  useTransferStore.getState().updateTransfer(item.id, {
    tempUri: tempPath,
  });
  await FileSystem.copyAsync({ from: item.uri, to: tempPath });
  return { uploadUri: tempPath, isTemp: true };
}

async function cleanupTempFile(item: TransferItem): Promise<void> {
  const tempUri = item.tempUri;
  if (!tempUri) return;
  try {
    const info = await FileSystem.getInfoAsync(tempUri);
    if (info.exists) {
      await FileSystem.deleteAsync(tempUri, { idempotent: true });
    }
  } catch {}
}

function createUploadTask(
  item: TransferItem,
  uploadUri: string,
  onProgress: (progress: {
    totalBytesSent: number;
    totalBytesExpectedToSend: number;
  }) => void
): FileSystem.UploadTask {
  return FileSystem.createUploadTask(
    httpUrl(item.agentAddress, "/transfers/upload"),
    uploadUri,
    {
      httpMethod: "POST",
      uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
      headers: {
        Authorization: `Bearer ${item.authToken}`,
        "X-Filename": item.filename,
        "X-Transfer-Id": item.id,
      },
    },
    onProgress
  );
}

export function enqueueFiles(
  files: { uri: string; name: string; size: number }[],
  agentAddress: string,
  authToken: string
): number {
  const items: TransferItem[] = files.map((f) => ({
    id: createId("transfer"),
    uri: f.uri,
    filename: f.name,
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
    if (activeTask) {
      activeTask.cancelAsync();
      activeTask = null;
    }
    useTransferStore.getState().updateTransfer(id, {
      status: "cancelled",
      completedAt: Date.now(),
    });
    cleanupTempFile(t);
  } else {
    useTransferStore.getState().updateTransfer(id, {
      status: "cancelled",
      completedAt: Date.now(),
    });
    cleanupTempFile(t);
  }
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
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  const active = getActiveUpload();
  if (active && active.id === id) {
    intentionallyCancelled = true;
    if (activeTask) {
      activeTask.cancelAsync();
      activeTask = null;
    }
  }
  useTransferStore.getState().removeTransfer(id);
  if (t) cleanupTempFile(t);
}

export function stopAll(): void {
  const store = useTransferStore.getState();
  const active = getActiveUpload();

  if (active) {
    intentionallyCancelled = true;
    if (activeTask) {
      activeTask.cancelAsync();
      activeTask = null;
    }
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
  const completed = useTransferStore
    .getState()
    .transfers.filter((t) => t.status === "completed");
  for (const t of completed) {
    cleanupTempFile(t);
  }
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

async function runStreamingUpload(item: TransferItem): Promise<void> {
  const startTime = Date.now();
  intentionallyCancelled = false;

  let uploadUri = item.uri;
  let isTemp = false;

  // Files are picked with copyToCacheDirectory:true, so URIs are file://.
  // Defensive: if a content:// URI slips through, copy it to a file:// cache
  // path first — the native uploader cannot read content:// URIs (that left
  // uploads stuck at 0%).
  if (isContentUri(item.uri)) {
    try {
      const prepared = await copyToCache(item);
      uploadUri = prepared.uploadUri;
      isTemp = true;
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to prepare file";
      useTransferStore.getState().updateTransfer(item.id, {
        status: "failed",
        error: msg,
        completedAt: Date.now(),
      });
      return;
    }
  }

  let lastUpdateTime = 0;
  let lastProgress = 0;

  const onProgress = (progress: {
    totalBytesSent: number;
    totalBytesExpectedToSend: number;
  }) => {
    const now = Date.now();
    const sentBytes = progress.totalBytesSent;
    const expected = progress.totalBytesExpectedToSend;
    const total = expected > 0 ? expected : item.size;
    const p = total > 0 ? sentBytes / total : 0;

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

  const task = createUploadTask(item, uploadUri, onProgress);
  activeTask = task;

  try {
    const result = await task.uploadAsync();
    activeTask = null;

    if (intentionallyCancelled) {
      if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
      return;
    }

    if (!result) {
      useTransferStore.getState().updateTransfer(item.id, {
        status: "cancelled",
        completedAt: Date.now(),
      });
      if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
      return;
    }

    const json = JSON.parse(result.body) as {
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

    if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
  } catch (err) {
    activeTask = null;
    if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
    if (intentionallyCancelled) return;

    const msg = err instanceof Error ? err.message : "Upload failed";
    useTransferStore.getState().updateTransfer(item.id, {
      status: "failed",
      error: msg,
      completedAt: Date.now(),
    });
  }
}
