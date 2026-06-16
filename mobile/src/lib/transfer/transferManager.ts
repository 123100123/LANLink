import * as FileSystem from "expo-file-system/legacy";
import { httpUrl } from "@/lib/api/endpoints";
import { createId } from "@/lib/protocol/envelope";
import { useTransferStore, type TransferItem } from "@/store/transferStore";

const CHUNK_SIZE = 1024 * 1024;

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
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

async function prepareFileForUpload(
  item: TransferItem
): Promise<{ uploadUri: string; isTemp: boolean }> {
  if (!isContentUri(item.uri)) {
    return { uploadUri: item.uri, isTemp: false };
  }

  const freeSpace = await getFreeDiskSpace();
  if (freeSpace >= 0 && item.size > 0 && freeSpace < item.size + 1024 * 1024) {
    throw new Error(
      `Not enough disk space. Need ${formatSize(item.size)}, only ${formatSize(freeSpace)} available.`
    );
  }

  const cacheDir = FileSystem.cacheDirectory ?? "";
  const tempPath = `${cacheDir}upload_${item.id}_${item.filename}`;
  await FileSystem.copyAsync({ from: item.uri, to: tempPath });
  useTransferStore.getState().updateTransfer(item.id, { tempUri: tempPath });
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

function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

async function readChunk(
  uri: string,
  offset: number,
  length: number
): Promise<ArrayBuffer> {
  const base64 = await FileSystem.readAsStringAsync(uri, {
    encoding: FileSystem.EncodingType.Base64,
    position: offset,
    length,
  });
  return base64ToArrayBuffer(base64);
}

async function apiStart(
  address: string,
  token: string,
  transferId: string,
  filename: string,
  size: number
): Promise<{ received: number }> {
  const resp = await fetch(httpUrl(address, "/transfers/resumable/start"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ transfer_id: transferId, filename, size }),
  });
  const json = (await resp.json()) as any;
  if (!resp.ok) throw new Error(json.error || "Failed to start transfer");
  return { received: json.received ?? 0 };
}

async function apiStatus(
  address: string,
  token: string,
  transferId: string
): Promise<{ received: number; total: number }> {
  const resp = await fetch(
    httpUrl(
      address,
      `/transfers/resumable/${encodeURIComponent(transferId)}/status`
    ),
    { headers: { Authorization: `Bearer ${token}` } }
  );
  const json = (await resp.json()) as any;
  if (!resp.ok) throw new Error(json.error || "Failed to get status");
  return { received: json.received ?? 0, total: json.total ?? 0 };
}

async function apiChunk(
  address: string,
  token: string,
  transferId: string,
  offset: number,
  data: ArrayBuffer
): Promise<{ received: number; total: number }> {
  const resp = await fetch(
    httpUrl(
      address,
      `/transfers/resumable/${encodeURIComponent(transferId)}/chunk?offset=${offset}`
    ),
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/octet-stream",
        Authorization: `Bearer ${token}`,
      },
      body: data,
    }
  );
  const json = (await resp.json()) as any;
  if (resp.status === 409) {
    const expected = json.expected_offset ?? json.received ?? 0;
    throw { offsetMismatch: true, expected };
  }
  if (!resp.ok) throw new Error(json.error || "Chunk failed");
  return { received: json.received ?? 0, total: json.total ?? 0 };
}

async function apiFinish(
  address: string,
  token: string,
  transferId: string
): Promise<{ path: string }> {
  const resp = await fetch(
    httpUrl(
      address,
      `/transfers/resumable/${encodeURIComponent(transferId)}/finish`
    ),
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  const json = (await resp.json()) as any;
  if (!resp.ok) throw new Error(json.error || "Finish failed");
  return { path: json.path ?? "" };
}

async function apiCancel(
  address: string,
  token: string,
  transferId: string
): Promise<void> {
  try {
    await fetch(
      httpUrl(
        address,
        `/transfers/resumable/${encodeURIComponent(transferId)}`
      ),
      {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      }
    );
  } catch {}
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
    sentBytes: 0,
    progress: 0,
    status: "waiting" as const,
    speed: 0,
    elapsed: 0,
    startedAt: Date.now(),
  }));

  useTransferStore.getState().addTransfers(items);
  processQueue(agentAddress, authToken);
  return items.length;
}

export function pauseTransfer(
  id: string,
  _agentAddress?: string,
  _authToken?: string
): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  if (!t) return;

  const active = getActiveUpload();
  if (active && active.id === id) {
    intentionallyCancelled = true;
    useTransferStore.getState().updateTransfer(id, { status: "paused" });
  } else if (t.status === "waiting") {
    useTransferStore.getState().updateTransfer(id, { status: "paused" });
  }
}

export function resumeTransfer(
  id: string,
  agentAddress: string,
  authToken: string
): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  if (!t || t.status !== "paused") return;
  useTransferStore.getState().updateTransfer(id, { status: "waiting" });
  processQueue(agentAddress, authToken);
}

export function cancelTransfer(
  id: string,
  agentAddress?: string,
  authToken?: string
): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  if (!t) return;

  const active = getActiveUpload();
  if (active && active.id === id) {
    intentionallyCancelled = true;
    useTransferStore.getState().updateTransfer(id, {
      status: "cancelled",
      completedAt: Date.now(),
    });
    if (agentAddress && authToken) {
      apiCancel(agentAddress, authToken, t.id);
    }
    cleanupTempFile(t);
  } else {
    useTransferStore.getState().updateTransfer(id, {
      status: "cancelled",
      completedAt: Date.now(),
    });
    if (agentAddress && authToken) {
      apiCancel(agentAddress, authToken, t.id);
    }
    cleanupTempFile(t);
  }
}

export function retryTransfer(
  id: string,
  agentAddress: string,
  authToken: string
): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  if (!t || t.status !== "failed") return;
  if (agentAddress && authToken) {
    apiCancel(agentAddress, authToken, t.id);
  }
  useTransferStore.getState().updateTransfer(id, {
    status: "waiting",
    error: undefined,
    sentBytes: 0,
    progress: 0,
    speed: 0,
    elapsed: 0,
  });
  processQueue(agentAddress, authToken);
}

export function removeTransfer(
  id: string,
  agentAddress?: string,
  authToken?: string
): void {
  const t = useTransferStore.getState().transfers.find((x) => x.id === id);
  const active = getActiveUpload();
  if (active && active.id === id) {
    intentionallyCancelled = true;
    if (agentAddress && authToken && t) {
      apiCancel(agentAddress, authToken, t.id);
    }
  }
  useTransferStore.getState().removeTransfer(id);
  if (t) cleanupTempFile(t);
}

export function pauseAll(): void {
  const store = useTransferStore.getState();
  const active = getActiveUpload();

  if (active) {
    intentionallyCancelled = true;
    store.updateTransfer(active.id, { status: "paused" });
  }

  for (const t of store.transfers) {
    if (t.status === "waiting") {
      store.updateTransfer(t.id, { status: "paused" });
    }
  }
}

export function resumeAll(agentAddress: string, authToken: string): void {
  const store = useTransferStore.getState();
  for (const t of store.transfers) {
    if (t.status === "paused") {
      store.updateTransfer(t.id, { status: "waiting" });
    }
  }
  processQueue(agentAddress, authToken);
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

async function processQueue(
  agentAddress: string,
  authToken: string
): Promise<void> {
  if (queueLoopRunning) return;
  if (getActiveUpload()) return;

  queueLoopRunning = true;

  try {
    while (true) {
      const next = getNextWaiting();
      if (!next) break;
      useTransferStore
        .getState()
        .updateTransfer(next.id, { status: "uploading" });
      await runResumableUpload(next, agentAddress, authToken);
    }
  } finally {
    queueLoopRunning = false;
  }
}

async function runResumableUpload(
  item: TransferItem,
  agentAddress: string,
  authToken: string
): Promise<void> {
  const startTime = Date.now();
  intentionallyCancelled = false;

  let uploadUri = item.uri;
  let isTemp = false;

  try {
    const prepared = await prepareFileForUpload(item);
    uploadUri = prepared.uploadUri;
    isTemp = prepared.isTemp;
  } catch (err) {
    const msg = err instanceof Error ? err.message : "Failed to prepare file";
    useTransferStore.getState().updateTransfer(item.id, {
      status: "failed",
      error: msg,
      completedAt: Date.now(),
    });
    return;
  }

  try {
    await apiStart(
      agentAddress,
      authToken,
      item.id,
      item.filename,
      item.size
    );
  } catch (err) {
    const msg =
      err instanceof Error ? err.message : "Failed to start transfer";
    useTransferStore.getState().updateTransfer(item.id, {
      status: "failed",
      error: msg,
      completedAt: Date.now(),
    });
    if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
    return;
  }

  let offset = 0;
  try {
    const status = await apiStatus(agentAddress, authToken, item.id);
    offset = status.received;
  } catch {}

  try {
    while (offset < item.size) {
      if (intentionallyCancelled) break;

      const toRead = Math.min(CHUNK_SIZE, item.size - offset);
      const ab = await readChunk(uploadUri, offset, toRead);

      if (intentionallyCancelled) break;

      try {
        const result = await apiChunk(
          agentAddress,
          authToken,
          item.id,
          offset,
          ab
        );
        offset = result.received;
      } catch (err: any) {
        if (err?.offsetMismatch) {
          offset = err.expected;
          continue;
        }
        throw err;
      }

      const elapsed = (Date.now() - startTime) / 1000;
      const speed = elapsed > 0 ? offset / elapsed : 0;

      useTransferStore.getState().updateTransfer(item.id, {
        sentBytes: offset,
        size: item.size,
        progress: item.size > 0 ? offset / item.size : 0,
        speed,
        elapsed,
        status: "uploading",
      });
    }

    if (intentionallyCancelled) {
      if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
      return;
    }

    const result = await apiFinish(agentAddress, authToken, item.id);

    useTransferStore.getState().updateTransfer(item.id, {
      status: "completed",
      progress: 1,
      sentBytes: item.size,
      completedAt: Date.now(),
      savedPath: result.path,
    });

    if (isTemp) cleanupTempFile({ ...item, tempUri: uploadUri });
  } catch (err) {
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
