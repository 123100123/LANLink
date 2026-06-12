import { useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";

import { checkHealth } from "@/lib/api/http";
import { useDevicesQuery } from "@/hooks/useDevices";
import { useHealth } from "@/hooks/useHealth";
import { usePing } from "@/hooks/usePing";
import { useSessionStore } from "@/store/sessionStore";

export default function HomeScreen() {
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const [targetId, setTargetId] = useState("");
  const [healthStatus, setHealthStatus] = useHealth();
  const devices = useDevicesQuery();
  const ping = usePing();

  const activeDevice = useMemo(() => devices.data?.devices.find((device) => device.device_id === targetId), [devices.data, targetId]);

  async function refreshHealth() {
    setHealthStatus({ loading: true, message: "Checking..." });
    try {
      const result = await checkHealth(agentAddress);
      setHealthStatus({ loading: false, message: `${result.status} / ${result.service}` });
    } catch (error) {
      setHealthStatus({ loading: false, message: error instanceof Error ? error.message : "Health check failed" });
    }
  }

  return (
    <ScrollView
      contentContainerStyle={styles.container}
      refreshControl={<RefreshControl refreshing={devices.isFetching} onRefresh={() => devices.refetch()} />}
    >
      <Text style={styles.title}>LANLink</Text>
      <Text style={styles.subtitle}>Connected tools for your local agent.</Text>

      <View style={styles.card}>
        <Text style={styles.section}>Agent</Text>
        <Text style={styles.body}>{agentAddress || "No agent address saved yet"}</Text>
        <Pressable style={styles.button} onPress={refreshHealth}>
          {healthStatus.loading ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Health check</Text>}
        </Pressable>
        <Text style={styles.status}>{healthStatus.message}</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.section}>Devices</Text>
        <Text style={styles.body}>{devices.data?.devices.length ? `${devices.data.devices.length} device(s) available` : "No paired devices yet"}</Text>
        <TextInput
          value={targetId}
          onChangeText={setTargetId}
          placeholder="device id for ping/messaging"
          placeholderTextColor="#5f6f8f"
          style={styles.input}
        />
        <Pressable
          style={styles.secondaryButton}
          onPress={() => activeDevice && ping.setTarget(activeDevice.device_id, activeDevice.device_name)}
          disabled={!activeDevice}
        >
          <Text style={styles.secondaryButtonText}>Use selected device</Text>
        </Pressable>
      </View>

      <View style={styles.card}>
        <Text style={styles.section}>Ping</Text>
        <Text style={styles.body}>
          {ping.status ? `${ping.status}${typeof ping.latencyMs === "number" ? ` - ${ping.latencyMs.toFixed(1)} ms` : ""}` : "Pick a device and ping it."}
        </Text>
        <Pressable style={styles.button} onPress={() => void ping.runPing()} disabled={!ping.canPing}>
          <Text style={styles.buttonText}>Run ping</Text>
        </Pressable>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flexGrow: 1,
    padding: 20,
    backgroundColor: "#0b1220",
  },
  title: {
    color: "#fff",
    fontSize: 32,
    fontWeight: "800",
    marginTop: 24,
  },
  subtitle: {
    color: "#b6c2d6",
    marginTop: 8,
    marginBottom: 20,
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 20,
    padding: 18,
    marginBottom: 16,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  section: {
    color: "#fff",
    fontSize: 18,
    fontWeight: "700",
    marginBottom: 8,
  },
  body: {
    color: "#b6c2d6",
    lineHeight: 20,
  },
  button: {
    marginTop: 14,
    backgroundColor: "#4f7cff",
    paddingVertical: 14,
    alignItems: "center",
    borderRadius: 14,
  },
  buttonText: {
    color: "#fff",
    fontWeight: "700",
  },
  secondaryButton: {
    marginTop: 14,
    backgroundColor: "#19253d",
    paddingVertical: 14,
    alignItems: "center",
    borderRadius: 14,
  },
  secondaryButtonText: {
    color: "#fff",
    fontWeight: "700",
  },
  status: {
    color: "#9db1d1",
    marginTop: 12,
  },
  input: {
    backgroundColor: "#09101d",
    borderRadius: 14,
    paddingHorizontal: 14,
    paddingVertical: 12,
    color: "#fff",
    borderWidth: 1,
    borderColor: "#23324f",
    marginTop: 14,
  },
});
