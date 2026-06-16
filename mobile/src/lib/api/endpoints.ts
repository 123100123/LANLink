export function normalizeAddress(address: string): string {
  return address.trim().replace(/^https?:\/\//i, "").replace(/^ws?:\/\//i, "").replace(/\/+$/, "");
}

export function httpUrl(address: string, path: string): string {
  return `http://${normalizeAddress(address)}${path}`;
}

export function wsUrl(address: string): string {
  return `ws://${normalizeAddress(address)}/ws`;
}
