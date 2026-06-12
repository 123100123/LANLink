import type { LanLinkMessage } from "@/lib/protocol/payloads";

type Listener = (message: LanLinkMessage) => void;

export class MessageRouter {
  private listeners = new Set<Listener>();

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  emit(message: LanLinkMessage) {
    for (const listener of this.listeners) {
      listener(message);
    }
  }
}
