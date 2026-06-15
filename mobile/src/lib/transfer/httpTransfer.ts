import type {
  TransferStartRequest,
  TransferStartResponse,
  TransferChunkResponse,
  TransferFinishResponse,
} from "@/lib/protocol/payloads";
import { httpUrl } from "@/lib/api/endpoints";
import { createId } from "@/lib/protocol/envelope";
import { getExpoFile, getFileSize, readChunk, CHUNK_SIZE, type PickedFile } from "./chunker";

const DEFAULT_CONCURRENCY = 6;

export type TransferProgress = {
  transferId: string;
  filename: string;
  sentBytes: number;
  totalBytes: number;
  progress: number;
  speed: number;
  elapsed: number;
};

export type TransferOptions = {
  transferId?: string;
  concurrency?: number;
  chunkSize?: number;
  signal?: AbortSignal;
  onProgress?: (progress: TransferProgress) => void;
};

export type TransferResult = {
  transferId: string;
  path: string;
  received: number;
  total: number;
};

export async function httpTransfer(
  address: string,
  authToken: string,
  file: PickedFile,
  options: TransferOptions = {}
): Promise<TransferResult> {
  const transferId = options.transferId ?? createId("transfer");
  const concurrency = options.concurrency ?? DEFAULT_CONCURRENCY;
  const chunkSize = options.chunkSize ?? CHUNK_SIZE;

  const expoFile = getExpoFile(file.uri);
  const totalSize = file.size ?? getFileSize(expoFile);

  await httpStartTransfer(address, authToken, transferId, file.name, totalSize);

  const startTime = Date.now();
  let sentBytes = 0;

  const totalChunks = Math.ceil(totalSize / chunkSize);
  let nextIndex = 0;
  let completedChunks = 0;
  let failed = false;
  let firstError: Error | null = null;

  const workers = Array.from({ length: Math.min(concurrency, totalChunks) }, () =>
    (async () => {
      while (!failed && nextIndex < totalChunks) {
        if (options.signal?.aborted) {
          failed = true;
          firstError ??= new Error("Upload cancelled");
          return;
        }

        const index = nextIndex++;
        const offset = index * chunkSize;
        const length = Math.min(chunkSize, totalSize - offset);

        try {
          const data = readChunk(expoFile, offset, length);
          await httpSendChunk(address, authToken, transferId, index, offset, data);
        } catch (err) {
          failed = true;
          firstError ??= err instanceof Error ? err : new Error("Chunk upload failed");
          return;
        }

        completedChunks++;
        sentBytes += length;

        const elapsed = (Date.now() - startTime) / 1000;
        const speed = elapsed > 0 ? sentBytes / elapsed : 0;

        options.onProgress?.({
          transferId,
          filename: file.name,
          sentBytes,
          totalBytes: totalSize,
          progress: totalSize > 0 ? sentBytes / totalSize : 0,
          speed,
          elapsed,
        });
      }
    })()
  );

  await Promise.all(workers);

  if (failed) {
    throw firstError ?? new Error("Upload failed");
  }

  const finishResult = await httpFinishTransfer(address, authToken, transferId);

  return {
    transferId,
    path: finishResult.path ?? "",
    received: finishResult.received ?? totalSize,
    total: finishResult.total ?? totalSize,
  };
}

export function createTransferAbortController(): AbortController {
  return new AbortController();
}

async function httpStartTransfer(
  address: string,
  authToken: string,
  transferId: string,
  filename: string,
  size: number
): Promise<void> {
  const body: TransferStartRequest = {
    transfer_id: transferId,
    filename,
    size,
  };

  const response = await fetch(httpUrl(address, "/transfers/start"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${authToken}`,
    },
    body: JSON.stringify(body),
  });

  const json = (await response.json()) as TransferStartResponse;

  if (!response.ok || json.status !== "started") {
    throw new Error(json.error || `Failed to start transfer (${response.status})`);
  }
}

async function httpSendChunk(
  address: string,
  authToken: string,
  transferId: string,
  index: number,
  offset: number,
  data: Uint8Array
): Promise<void> {
  const encodedId = encodeURIComponent(transferId);
  const url = httpUrl(address, `/transfers/${encodedId}/chunks/${index}?offset=${offset}`);

  const response = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": "application/octet-stream",
      Authorization: `Bearer ${authToken}`,
    },
    body: data,
  });

  const json = (await response.json()) as TransferChunkResponse;

  if (!response.ok) {
    throw new Error(json.error || `Chunk ${index} failed (${response.status})`);
  }

  if (json.status !== "chunk.received" && json.status !== "chunk.duplicate") {
    throw new Error(`Chunk ${index} unexpected status: ${json.status}`);
  }
}

async function httpFinishTransfer(
  address: string,
  authToken: string,
  transferId: string
): Promise<TransferFinishResponse> {
  const encodedId = encodeURIComponent(transferId);
  const url = httpUrl(address, `/transfers/${encodedId}/finish`);

  const response = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${authToken}`,
    },
  });

  const json = (await response.json()) as TransferFinishResponse;

  if (!response.ok || json.status !== "saved") {
    throw new Error(json.error || `Failed to finish transfer (${response.status})`);
  }

  return json;
}
