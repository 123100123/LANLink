import * as FileSystem from "expo-file-system/legacy";

export const CHUNK_SIZE = 64 * 1024;

export type PickedFile = {
  uri: string;
  name: string;
  size?: number;
};

export async function getFileSize(uri: string): Promise<number> {
  const info = await FileSystem.getInfoAsync(uri);
  if (!info.exists || typeof info.size !== "number") {
    throw new Error("Unable to read file metadata");
  }
  return info.size;
}

export async function* readBase64Chunks(uri: string, chunkSize = CHUNK_SIZE): AsyncGenerator<{ index: number; content: string }> {
  let index = 0;
  let position = 0;
  const fileSize = await getFileSize(uri);

  while (position < fileSize) {
    const length = Math.min(chunkSize, fileSize - position);
    const content = await FileSystem.readAsStringAsync(uri, {
      encoding: "base64",
      position,
      length,
    });

    yield { index, content };
    index += 1;
    position += length;
  }
}
