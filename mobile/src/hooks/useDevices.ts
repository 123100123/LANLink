import { useQuery } from "@tanstack/react-query";

import { fetchDevices } from "@/lib/api/http";
import { useSessionStore } from "@/store/sessionStore";

export function useDevicesQuery() {
  const credentials = useSessionStore((state) => state.credentials);

  return useQuery({
    queryKey: ["devices", credentials?.deviceId, credentials?.agentAddress],
    enabled: Boolean(credentials?.authToken && credentials.agentAddress),
    queryFn: () => fetchDevices(credentials!.agentAddress, credentials!.authToken),
  });
}
