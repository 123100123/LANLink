import * as Network from "expo-network";

export type DiscoveredHost = {
  address: string; // host:port
  service: string;
};

// getSubnetBase returns the device's /24 prefix (e.g. "192.168.1.") or null.
// Mobile cannot listen for the UDP discovery beacon in managed Expo without a
// native module, so discovery uses an HTTP /health sweep of the local subnet.
export async function getSubnetBase(): Promise<string | null> {
  try {
    const ip = await Network.getIpAddressAsync();
    const parts = ip.split(".");
    if (parts.length !== 4) return null;
    return `${parts[0]}.${parts[1]}.${parts[2]}.`;
  } catch {
    return null;
  }
}

// sweepSubnet probes host .1–.254 of base on the given port for GET /health,
// returning reachable LANLink receivers. Probes run with bounded concurrency
// and a short per-host timeout; pass an AbortSignal to cancel early.
export async function sweepSubnet(
  base: string,
  port: number,
  onProgress?: (done: number, total: number) => void,
  signal?: AbortSignal,
): Promise<DiscoveredHost[]> {
  const total = 254;
  const concurrency = 40;
  const perHostTimeoutMs = 400;

  const hosts: DiscoveredHost[] = [];
  let done = 0;
  let next = 1;

  async function probe(i: number): Promise<void> {
    const address = `${base}${i}:${port}`;
    try {
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), perHostTimeoutMs);
      try {
        const resp = await fetch(`http://${address}/health`, { signal: controller.signal });
        if (resp.ok) {
          const data = (await resp.json()) as { service?: string };
          if (data && typeof data.service === "string") {
            hosts.push({ address, service: data.service });
          }
        }
      } finally {
        clearTimeout(timer);
      }
    } catch {
      // unreachable / timeout / non-LANLink host — ignore
    } finally {
      done += 1;
      onProgress?.(done, total);
    }
  }

  async function worker(): Promise<void> {
    while (true) {
      if (signal?.aborted) return;
      const i = next;
      next += 1;
      if (i > total) return;
      await probe(i);
    }
  }

  const workers: Promise<void>[] = [];
  for (let w = 0; w < concurrency; w += 1) {
    workers.push(worker());
  }
  await Promise.all(workers);

  return hosts;
}
