import { z } from "zod";

import type { LanLinkMessage } from "./payloads";

export const messageSchema = z.object({
  type: z.string(),
  id: z.string().optional(),
  module: z.string().optional(),
  action: z.string().optional(),
  timestamp: z.number().optional(),
  payload: z.any().optional(),
});

export function encodeMessage(message: LanLinkMessage): string {
  return JSON.stringify(message);
}

export function decodeMessage(raw: string): LanLinkMessage {
  return messageSchema.parse(JSON.parse(raw));
}

export function createId(prefix: string): string {
  const random = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}_${Math.random().toString(16).slice(2)}`;
  return `${prefix}_${random}`;
}
