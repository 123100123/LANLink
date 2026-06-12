import { create } from "zustand";

import { loadCredentials, saveCredentials } from "@/lib/storage/credentials";
import { loadPreferences, savePreferences } from "@/lib/storage/preferences";
import type { Credentials } from "@/lib/storage/credentials";

type SessionState = {
  hydrated: boolean;
  hasCredentials: boolean;
  agentAddress: string;
  credentials: Credentials | null;
  setAgentAddress: (address: string) => void;
  setCredentials: (credentials: Credentials) => Promise<void>;
  clearSession: () => void;
  hydrate: () => Promise<void>;
};

export const useSessionStore = create<SessionState>((set, get) => ({
  hydrated: false,
  hasCredentials: false,
  agentAddress: "",
  credentials: null,
  setAgentAddress: (address) => set({ agentAddress: address }),
  setCredentials: async (credentials) => {
    await saveCredentials(credentials);
    try {
      await savePreferences({
        ...(await loadPreferences()),
        agentAddress: credentials.agentAddress,
      });
    } catch {
      // Credentials are still valid even if preference persistence fails.
    }
    set({ credentials, hasCredentials: true, agentAddress: credentials.agentAddress });
  },
  clearSession: () => set({ credentials: null, hasCredentials: false }),
  hydrate: async () => {
    const [prefs, credentials] = await Promise.all([loadPreferences(), loadCredentials()]);
    set({
      hydrated: true,
      agentAddress: credentials?.agentAddress ?? prefs.agentAddress,
      credentials,
      hasCredentials: Boolean(credentials),
    });
  },
}));
