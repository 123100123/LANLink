import { useMemo } from "react";

import { LanLinkSocket } from "@/lib/socket/lanlinkSocket";
import { useSessionStore } from "@/store/sessionStore";

let singleton: LanLinkSocket | null = null;

export function useSocket() {
  const credentials = useSessionStore((state) => state.credentials);

  return useMemo(() => {
    if (!singleton) {
      singleton = new LanLinkSocket();
    }

    if (credentials) {
      singleton.setCredentials(credentials);
    }

    return singleton;
  }, [credentials]);
}
