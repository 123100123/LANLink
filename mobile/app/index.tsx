import { Redirect } from "expo-router";
import { View, Text, ActivityIndicator, StyleSheet } from "react-native";

import { useSessionStore } from "@/store/sessionStore";

export default function Index() {
  const hydrated = useSessionStore((state) => state.hydrated);
  const hasCredentials = useSessionStore((state) => state.hasCredentials);
  const credentials = useSessionStore((state) => state.credentials);

  if (!hydrated) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" color="#007AFF" />
        <Text style={styles.loadingText}>Loading LANLink...</Text>
      </View>
    );
  }

  if (!hasCredentials || !credentials?.deviceId) {
    return <Redirect href="/pair" />;
  }

  return <Redirect href="/(tabs)/device" />;
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    backgroundColor: "#fff",
  },
  loadingText: {
    marginTop: 16,
    fontSize: 16,
    color: "#666",
  },
});