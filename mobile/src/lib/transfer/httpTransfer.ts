import * as FileSystem from "expo-file-system/legacy";
import { httpUrl } from "@/lib/api/endpoints";
import { createId } from "@/lib/protocol/envelope";

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
  onProgress?: (progress: TransferProgress) => void;
};

export type TransferResult = {
  transferId: string;
  path: string;
  received: number;
  total: number;
};

let activeTask: FileSystem.UploadTask | null = null;
let activeStartTime = 0;

export async function httpTransfer(
  address: string,
  authToken: string,
  file: { uri: string; name: string; size?: number },
  options: TransferOptions = {}
): Promise<TransferResult> {
  const transferId = options.transferId ?? createId("transfer");
  const totalSize = file.size ?? 0;

  activeStartTime = Date.now();

  const task = FileSystem.createUploadTask(
    httpUrl(address, "/transfers/upload"),
    file.uri,
    {
      httpMethod: "POST",
      uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
      headers: {
        Authorization: `Bearer ${authToken}`,
        "X-Filename": file.name,
        "X-Transfer-Id": transferId,
      },
    },
    (progress) => {
      const sentBytes = progress.totalBytesSent;
      const expected = progress.totalBytesExpectedToSend;
      const total = expected > 0 ? expected : totalSize;
      const elapsed = (Date.now() - activeStartTime) / 1000;
      const speed = elapsed > 0 ? sentBytes / elapsed : 0;

      options.onProgress?.({
        transferId,
        filename: file.name,
        sentBytes,
        totalBytes: total,
        progress: total > 0 ? sentBytes / total : 0,
        speed,
        elapsed,
      });
    }
  );

  activeTask = task;

  try {
    const result = await task.uploadAsync();

    if (!result) {
      throw new Error("Upload cancelled");
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

    return {
      transferId,
      path: json.path ?? "",
      received: json.received ?? totalSize,
      total: totalSize,
    };
  } finally {
    activeTask = null;
  }
}

export async function cancelTransfer(): Promise<void> {
  if (activeTask) {
    await activeTask.cancelAsync();
    activeTask = null;
  }
}
