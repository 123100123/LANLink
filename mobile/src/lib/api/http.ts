import type { DevicesResponse, HealthResponse, PairRequest, PairResponse } from "@/lib/protocol/payloads";

import { httpUrl } from "./endpoints";

async function parseJson<T>(response: Response): Promise<T> {
  const data = (await response.json()) as T;
  return data;
}

// A fetch to a powered-off LAN host can hang for minutes on Android, which left
// the reachability badge stuck on "Checking…". Cap it so offline is reported fast.
const HEALTH_TIMEOUT_MS = 4000;

export async function checkHealth(address: string): Promise<HealthResponse> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), HEALTH_TIMEOUT_MS);
  let response: Response;
  try {
    response = await fetch(httpUrl(address, "/health"), {
      signal: controller.signal,
    });
  } catch (error) {
    if (controller.signal.aborted) {
      throw new Error("Health check timed out");
    }
    throw error;
  } finally {
    clearTimeout(timeout);
  }
  if (!response.ok) {
    throw new Error(`Health check failed (${response.status})`);
  }
  return parseJson<HealthResponse>(response);
}

export async function pairDevice(address: string, body: PairRequest): Promise<PairResponse> {
  const response = await fetch(httpUrl(address, "/pair"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  const json = await parseJson<PairResponse>(response);
  if (!response.ok || json.status !== "paired") {
    throw new Error(json.error || `Pairing failed (${response.status})`);
  }

  return json;
}

export async function fetchDevices(address: string, authToken: string): Promise<DevicesResponse> {
  const response = await fetch(httpUrl(address, "/devices"), {
    headers: {
      Authorization: `Bearer ${authToken}`,
    },
  });

  if (response.status === 401) {
    throw new Error("Unauthorized. Please pair again.");
  }

  if (!response.ok) {
    throw new Error(`Failed to load devices (${response.status})`);
  }

  return parseJson<DevicesResponse>(response);
}
