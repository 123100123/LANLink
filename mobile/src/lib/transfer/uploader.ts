import { createId } from "@/lib/protocol/envelope";
import type { FileChunkResponse } from "@/lib/protocol/payloads";
import { LanLinkSocket } from "@/lib/socket/lanlinkSocket";
import { getFileSize, readBase64Chunks, type PickedFile } from "./chunker";

export async function uploadFile(socket: LanLinkSocket, file: PickedFile): Promise<FileChunkResponse> {
  const transferId = createId("transfer");
  const size = file.size ?? (await getFileSize(file.uri));

  await socket.sendFileStart(transferId, file.name, size);

  for await (const chunk of readBase64Chunks(file.uri)) {
    await socket.sendFileChunk(transferId, chunk.index, chunk.content);
  }

  return socket.sendFileEnd(transferId);
}
