import { Redirect } from "expo-router";

import { useSessionStore } from "@/store/sessionStore";

export default function DevicesIndexRedirect() {
  const credentials = useSessionStore((state) => state.credentials);

  if (!credentials?.deviceId) {
    return <Redirect href="/setup" />;
  }

  return <Redirect href={`/(tabs)/devices/${credentials.deviceId}`} />;
}