import { Stack } from "expo-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { useEffect, useState } from "react";
import { View, Text, StyleSheet } from "react-native";

import { useSessionStore } from "@/store/sessionStore";

const queryClient = new QueryClient();

function ErrorBoundary({ error }: { error: Error }) {
  return (
    <View style={styles.errorContainer}>
      <Text style={styles.errorTitle}>⚠️ Error Loading App</Text>
      <Text style={styles.errorText}>{error.message}</Text>
      <Text style={styles.errorStack}>{error.stack}</Text>
    </View>
  );
}

export default function RootLayout() {
  const [error, setError] = useState<Error | null>(null);
  const hydrate = useSessionStore((state) => state.hydrate);

  useEffect(() => {
    console.log("[App] Starting hydrate...");
    hydrate()
      .then(() => {
        console.log("[App] Hydrate complete");
      })
      .catch((err) => {
        console.error("[App] Hydrate failed:", err);
        setError(err);
        useSessionStore.setState({ hydrated: true });
      });
  }, [hydrate]);

  if (error) {
    return <ErrorBoundary error={error} />;
  }

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <Stack screenOptions={{ headerShown: false }} />
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  errorContainer: {
    flex: 1,
    backgroundColor: "#f5f5f5",
    padding: 20,
    justifyContent: "center",
  },
  errorTitle: {
    fontSize: 18,
    fontWeight: "bold",
    color: "#d32f2f",
    marginBottom: 10,
  },
  errorText: {
    fontSize: 14,
    color: "#666",
    marginBottom: 10,
  },
  errorStack: {
    fontSize: 11,
    color: "#999",
    fontFamily: "monospace",
  },
});