import { Stack } from "expo-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import React, { useEffect, useState } from "react";
import { ScrollView, StyleSheet, Text } from "react-native";

import { useSessionStore } from "@/store/sessionStore";
import { useShareIntent } from "@/lib/share/useShareIntent";

const queryClient = new QueryClient();

function ErrorView({ title, error }: { title: string; error: Error }) {
  return (
    <ScrollView contentContainerStyle={styles.errorContainer}>
      <Text style={styles.errorTitle}>{title}</Text>
      <Text style={styles.errorText}>{error.message}</Text>
      <Text style={styles.errorStack}>{error.stack}</Text>
    </ScrollView>
  );
}

// Real React error boundary: catches render/runtime errors in any screen and
// shows them, instead of leaving a blank screen (which is what happens in a
// release build when an uncaught error escapes rendering).
class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { error: Error | null }
> {
  state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[App] Render error:", error, info?.componentStack);
  }

  render() {
    if (this.state.error) {
      return <ErrorView title="⚠️ App error" error={this.state.error} />;
    }
    return this.props.children;
  }
}

export default function RootLayout() {
  const [hydrateError, setHydrateError] = useState<Error | null>(null);
  const hydrate = useSessionStore((state) => state.hydrate);

  // Route files arriving from the Android system share sheet into the queue.
  useShareIntent();

  useEffect(() => {
    hydrate().catch((err) => {
      console.error("[App] Hydrate failed:", err);
      setHydrateError(err instanceof Error ? err : new Error(String(err)));
      // Let the app continue to the pairing screen even if storage failed.
      useSessionStore.setState({ hydrated: true });
    });
  }, [hydrate]);

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <ErrorBoundary>
          {hydrateError ? (
            <ErrorView title="⚠️ Error loading app" error={hydrateError} />
          ) : (
            <Stack screenOptions={{ headerShown: false }} />
          )}
        </ErrorBoundary>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  errorContainer: {
    flexGrow: 1,
    backgroundColor: "#0b1220",
    padding: 20,
    paddingTop: 60,
  },
  errorTitle: {
    fontSize: 18,
    fontWeight: "bold",
    color: "#ff8a8a",
    marginBottom: 12,
  },
  errorText: {
    fontSize: 14,
    color: "#d9e2f2",
    marginBottom: 12,
  },
  errorStack: {
    fontSize: 11,
    color: "#7d8aa5",
    fontFamily: "monospace",
  },
});
