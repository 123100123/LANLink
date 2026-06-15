import { File } from "expo-file-system";

export const CHUNK_SIZE = 128 * 1024;

export type PickedFile = {
  uri: string;
  name: string;
  size?: number;
};

export function getExpoFile(uri: string): File {
  return new File(uri);
}

export function getFileSize(file: File): number {
  return file.size;
}

export function readChunk(
  file: File,
  offset: number,
  length: number
): Uint8Array {
  const handle = file.open();
  try {
    handle.offset = offset;
    return handle.readBytes(length);
  } finally {
    handle.close();
  }
}
