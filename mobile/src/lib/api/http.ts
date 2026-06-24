import type { DevicesResponse, HealthResponse, PairRequest, PairResponse } from "@/lib/protocol/payloads";

import { httpUrl } from "./endpoints";

async function parseJson<T>(response: Response): Promise<T> {
  const data = (await response.json()) as T;
  return data;
}

export async function checkHealth(address: string): Promise<HealthResponse> {
  const response = await fetch(httpUrl(address, "/health"));
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

export async function pairAuto(address: string, deviceName: string): Promise<PairResponse> {
  const response = await fetch(httpUrl(address, "/pair/auto"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ device_name: deviceName }),
  });

  const json = await parseJson<PairResponse>(response);
  if (!response.ok || json.status !== "paired") {
    throw new Error(json.error || `Auto-connect failed (${response.status})`);
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
