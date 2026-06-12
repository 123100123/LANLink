import { Redirect } from "expo-router";
import { View, Text, ActivityIndicator, StyleSheet } from "react-native";

import { useSessionStore } from "@/store/sessionStore";

export default function Index() {
  const hydrated = useSessionStore((state) => state.hydrated);
  const hasCredentials = useSessionStore((state) => state.hasCredentials);

  if (!hydrated) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" color="#007AFF" />
        <Text style={styles.loadingText}>Loading LANLink...</Text>
      </View>
    );
  }

  return <Redirect href={hasCredentials ? "/(tabs)/home" : "/setup"} />;
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
