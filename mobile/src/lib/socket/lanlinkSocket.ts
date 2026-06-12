import { createId, decodeMessage } from "@/lib/protocol/envelope";
import type {
  AuthFailed,
  AuthRequest,
  AuthSuccess,
  DirectMessagePayload,
  DirectMessageResponse,
  FileChunkPayload,
  FileChunkResponse,
  FileEndPayload,
  FileStartPayload,
  PingPayload,
  PongPayload,
} from "@/lib/protocol/payloads";
import { wsUrl } from "@/lib/api/endpoints";
import type { Credentials } from "@/lib/storage/credentials";

type Resolver = {
  resolve: (value: { type: string; payload?: unknown }) => void;
  reject: (reason?: unknown) => void;
};

export class LanLinkSocket {
  private socket: WebSocket | null = null;
  private credentials: Credentials | null = null;
  private connected = false;
  private connecting: Promise<void> | null = null;
  private pending = new Map<string, Resolver>();
  private authPending: Resolver | null = null;

  setCredentials(credentials: Credentials) {
    this.credentials = credentials;
  }

  isConnected() {
    return this.connected;
  }

  async ensureConnected() {
    if (this.connected && this.socket?.readyState === WebSocket.OPEN) {
      return;
    }

    if (this.connecting) {
      return this.connecting;
    }

    if (!this.credentials) {
      throw new Error("No saved credentials. Pair the device first.");
    }

    this.connecting = this.connect();
    try {
      await this.connecting;
    } finally {
      this.connecting = null;
    }
  }

  private async connect() {
    const credentials = this.credentials;
    if (!credentials) {
      throw new Error("No credentials available.");
    }

    const socket = new WebSocket(wsUrl(credentials.agentAddress));
    this.socket = socket;

    socket.onmessage = (event) => {
      this.handleIncoming(String(event.data));
    };

    socket.onclose = () => {
      this.connected = false;
      this.failPending(new Error("WebSocket disconnected"));
    };

    await new Promise<void>((resolve, reject) => {
      socket.onopen = () => resolve();
      socket.onerror = () => reject(new Error("WebSocket connection failed"));
    });

    await this.authenticate();
  }

  private async authenticate() {
    const credentials = this.credentials;
    if (!credentials) {
      throw new Error("No credentials available.");
    }

    const response = await new Promise<{ type: string; payload?: unknown }>((resolve, reject) => {
      this.authPending = { resolve, reject };
      try {
        this.send({
          type: "auth",
          id: createId("auth"),
          timestamp: Date.now(),
          payload: {
            token: credentials.authToken,
          } satisfies AuthRequest,
        });
      } catch (error) {
        this.authPending = null;
        reject(error);
      }
    });

    if (response.type === "auth.failed") {
      const failed = response.payload as AuthFailed;
      this.close();
      throw new Error(failed.error);
    }

    if (response.type !== "auth.success") {
      this.close();
      throw new Error(`Unexpected auth response: ${response.type}`);
    }

    const success = response.payload as AuthSuccess;
    this.connected = true;
    return success;
  }

  private handleIncoming(raw: string) {
    const message = decodeMessage(raw);

    if (this.authPending && (message.type === "auth.success" || message.type === "auth.failed")) {
      const resolver = this.authPending;
      this.authPending = null;
      resolver.resolve({ type: message.type, payload: message.payload });
      return;
    }

    if (message.id && this.pending.has(message.id)) {
      const resolver = this.pending.get(message.id)!;
      this.pending.delete(message.id);
      resolver.resolve({ type: message.type, payload: message.payload });
    }
  }

  private failPending(error: Error) {
    if (this.authPending) {
      this.authPending.reject(error);
      this.authPending = null;
    }

    for (const resolver of this.pending.values()) {
      resolver.reject(error);
    }
    this.pending.clear();
  }

  private send(message: unknown) {
    const socket = this.socket;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket is not connected");
    }

    socket.send(JSON.stringify(message));
  }

  private sendAndWait(type: string, payload: unknown, idPrefix = type): Promise<{ type: string; payload?: unknown }> {
    const id = createId(idPrefix);
    const message = {
      type,
      id,
      timestamp: Date.now(),
      payload,
    };

    return new Promise<{ type: string; payload?: unknown }>((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      try {
        this.send(message);
      } catch (error) {
        this.pending.delete(id);
        reject(error);
      }
    });
  }

  async sendPing(sentAt = Date.now()): Promise<PongPayload> {
    await this.ensureConnected();
    const response = await this.sendAndWait("ping", { sent_at: sentAt } satisfies PingPayload, "ping");

    if (response.type !== "pong") {
      throw new Error(`Unexpected ping response: ${response.type}`);
    }

    return response.payload as PongPayload;
  }

  async sendDirectMessage(text: string): Promise<DirectMessageResponse> {
    await this.ensureConnected();
    const response = await this.sendAndWait("direct_message", { text } satisfies DirectMessagePayload, "message");

    if (response.type !== "direct_message.response") {
      throw new Error(`Unexpected message response: ${response.type}`);
    }

    return response.payload as DirectMessageResponse;
  }

  async sendFileStart(transferId: string, filename: string, size: number): Promise<FileChunkResponse> {
    await this.ensureConnected();
    const response = await this.sendAndWait(
      "file.start",
      { transfer_id: transferId, filename, size } satisfies FileStartPayload,
      "file_start"
    );

    if (response.type !== "file.chunk.response") {
      throw new Error(`Unexpected file start response: ${response.type}`);
    }

    return response.payload as FileChunkResponse;
  }

  async sendFileChunk(transferId: string, index: number, content: string): Promise<FileChunkResponse> {
    const response = await this.sendAndWait(
      "file.chunk",
      { transfer_id: transferId, index, content } satisfies FileChunkPayload,
      `file_chunk_${index}`
    );

    if (response.type !== "file.chunk.response") {
      throw new Error(`Unexpected file chunk response: ${response.type}`);
    }

    return response.payload as FileChunkResponse;
  }

  async sendFileEnd(transferId: string): Promise<FileChunkResponse> {
    const response = await this.sendAndWait("file.end", { transfer_id: transferId } satisfies FileEndPayload, "file_end");

    if (response.type !== "file.chunk.response") {
      throw new Error(`Unexpected file end response: ${response.type}`);
    }

    return response.payload as FileChunkResponse;
  }

  close() {
    this.socket?.close();
    this.socket = null;
    this.connected = false;
    this.failPending(new Error("WebSocket closed"));
  }
}
