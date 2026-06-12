import { createId } from "@/lib/protocol/envelope";
import type { FileChunkResponse } from "@/lib/protocol/payloads";
import { LanLinkSocket } from "@/lib/socket/lanlinkSocket";
import { getFileSize, readBase64Chunks, type PickedFile } from "./chunker";

export type UploadProgress = {
  transferId: string;
  filename: string;
  sentBytes: number;
  totalBytes: number;
  progress: number;
  chunkIndex: number;
};

export type UploadFileOptions = {
  transferId?: string;
  onProgress?: (progress: UploadProgress) => void;
};

export async function uploadFile(
  socket: LanLinkSocket,
  file: PickedFile,
  options: UploadFileOptions = {}
): Promise<FileChunkResponse> {
  const transferId = options.transferId ?? createId("transfer");
  const size = file.size ?? (await getFileSize(file.uri));

  await socket.sendFileStart(transferId, file.name, size);

  for await (const chunk of readBase64Chunks(file.uri)) {
    await socket.sendFileChunk(transferId, chunk.index, chunk.content);

    options.onProgress?.({
      transferId,
      filename: file.name,
      sentBytes: chunk.sentBytes,
      totalBytes: size,
      progress: size > 0 ? chunk.sentBytes / size : chunk.progress,
      chunkIndex: chunk.index,
    });
  }

  return socket.sendFileEnd(transferId);
}