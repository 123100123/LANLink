import { Link } from "expo-router";
import { ActivityIndicator, FlatList, Pressable, StyleSheet, Text, View } from "react-native";

import { useDevicesQuery } from "@/hooks/useDevices";
import { useSessionStore } from "@/store/sessionStore";

export default function DevicesScreen() {
  const query = useDevicesQuery();
  const agentAddress = useSessionStore((state) => state.agentAddress);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Devices</Text>
      <Text style={styles.subtitle}>{agentAddress || "No agent configured"}</Text>

      {query.isLoading ? (
        <ActivityIndicator color="#fff" />
      ) : (
        <FlatList
          data={query.data?.devices ?? []}
          keyExtractor={(item) => item.device_id}
          contentContainerStyle={styles.list}
          renderItem={({ item }) => (
            <Link href={`/(tabs)/devices/${item.device_id}`} asChild>
              <Pressable style={styles.card}>
                <Text style={styles.deviceName}>{item.device_name}</Text>
                <Text style={styles.deviceId}>{item.device_id}</Text>
              </Pressable>
            </Link>
          )}
          ListEmptyComponent={<Text style={styles.empty}>No devices returned yet.</Text>}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
    backgroundColor: "#0b1220",
  },
  title: {
    color: "#fff",
    fontSize: 30,
    fontWeight: "800",
    marginTop: 24,
  },
  subtitle: {
    color: "#b6c2d6",
    marginTop: 8,
    marginBottom: 16,
  },
  list: {
    paddingBottom: 24,
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 18,
    padding: 16,
    marginBottom: 12,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  deviceName: {
    color: "#fff",
    fontSize: 16,
    fontWeight: "700",
  },
  deviceId: {
    color: "#9db1d1",
    marginTop: 6,
  },
  empty: {
    color: "#9db1d1",
    marginTop: 20,
  },
});
