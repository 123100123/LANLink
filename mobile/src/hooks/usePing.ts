import { useState } from "react";

import type { PongPayload } from "@/lib/protocol/payloads";
import { useSessionStore } from "@/store/sessionStore";
import { useSocket } from "./useSocket";

export function usePing() {
  const credentials = useSessionStore((state) => state.credentials);
  const socket = useSocket();
  const [targetDeviceId, setTargetDeviceId] = useState("");
  const [targetDeviceName, setTargetDeviceName] = useState("");
  const [status, setStatus] = useState<string>("");
  const [latencyMs, setLatencyMs] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const canPing = Boolean(credentials?.authToken && credentials.agentAddress && targetDeviceId);

  async function runPing(overrideTargetId?: string) {
    const sentAt = Date.now();
    setError(null);
    setStatus("Sending ping...");
    await socket.ensureConnected();
    const payload = await socket.sendPing(sentAt);
    const result = payload as PongPayload;
    setLatencyMs(Math.max(0, result.received_at - result.sent_at));
    setStatus(`Pinged ${(overrideTargetId ?? targetDeviceId) || "device"}`);
    return result;
  }

  function setTarget(deviceId: string, deviceName: string) {
    setTargetDeviceId(deviceId);
    setTargetDeviceName(deviceName);
    setStatus(`Target set to ${deviceName || deviceId}`);
  }

  function setResult(result: PongPayload) {
    setLatencyMs(Math.max(0, result.received_at - result.sent_at));
    setStatus("Ping complete");
  }

  function setErrorMessage(message: string) {
    setError(message);
    setStatus(message);
  }

  return {
    canPing,
    latencyMs,
    error,
    status: error ?? status,
    targetDeviceId,
    targetDeviceName,
    setTarget,
    setResult,
    setError: setErrorMessage,
    runPing,
  };
}
