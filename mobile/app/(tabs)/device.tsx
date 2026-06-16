import * as DocumentPicker from "expo-document-picker";
import { useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";

import { useDevicesQuery } from "@/hooks/useDevices";
import { usePing } from "@/hooks/usePing";
import { useSocket } from "@/hooks/useSocket";
import { useSessionStore } from "@/store/sessionStore";
import { enqueueFiles } from "@/lib/transfer/transferManager";

export default function DeviceScreen() {
  const credentials = useSessionStore((state) => state.credentials);
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const devices = useDevicesQuery();
  const socket = useSocket();
  const ping = usePing();

  const [busy, setBusy] = useState(false);
  const [pingStatus, setPingStatus] = useState("");
  const [pickStatus, setPickStatus] = useState("");

  const device = devices.data?.devices.find(
    (d) => d.device_id === credentials?.deviceId
  );

  async function handlePing() {
    setBusy(true);
    setPingStatus("Pinging...");
    try {
      await socket.ensureConnected();
      const result = await ping.runPing();
      ping.setResult(result);
      setPingStatus(`${ping.latencyMs?.toFixed(1)} ms`);
    } catch (error) {
      const msg = error instanceof Error ? error.message : "Ping failed";
      ping.setError(msg);
      setPingStatus(msg);
    } finally {
      setBusy(false);
    }
  }

  async function handleSendFile() {
    try {
      const result = await DocumentPicker.getDocumentAsync({
        copyToCacheDirectory: false,
        multiple: true,
      });

      if (result.canceled || result.assets.length === 0) {
        return;
      }

      const count = enqueueFiles(
        result.assets.map((a) => ({
          uri: a.uri,
          name: a.name ?? "unknown file",
          size: a.size ?? 0,
        })),
        agentAddress!,
        credentials!.authToken
      );

      setPickStatus(`${count} file${count > 1 ? "s" : ""} added to queue`);
    } catch (error) {
      const msg = error instanceof Error ? error.message : "Failed to pick files";
      setPickStatus(msg);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Device</Text>

      <View style={styles.card}>
        <Text style={styles.label}>Device name</Text>
        <Text style={styles.value}>{device?.device_name ?? "Loading..."}</Text>

        <Text style={styles.label}>Device ID</Text>
        <Text style={styles.valueSmall}>{credentials?.deviceId ?? "\u2014"}</Text>

        <Text style={styles.label}>Agent</Text>
        <Text style={styles.valueSmall}>{agentAddress || "\u2014"}</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.section}>Ping</Text>
        <Text style={styles.body}>
          {pingStatus || "Test latency to the agent."}
        </Text>
        <Pressable style={styles.button} onPress={handlePing} disabled={busy}>
          {busy ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Ping</Text>
          )}
        </Pressable>
      </View>

      <View style={styles.card}>
        <Text style={styles.section}>Send file</Text>
        <Text style={styles.body}>
          Pick one or more files to upload to the agent.
        </Text>
        <Pressable style={styles.secondaryButton} onPress={handleSendFile}>
          <Text style={styles.buttonText}>Send file</Text>
        </Pressable>
        {pickStatus ? (
          <Text style={styles.status}>{pickStatus}</Text>
        ) : null}
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
    fontSize: 30,
    fontWeight: "800",
    marginTop: 24,
    marginBottom: 16,
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 20,
    padding: 18,
    marginBottom: 16,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  label: {
    color: "#d9e2f2",
    fontWeight: "600",
    marginBottom: 4,
    marginTop: 10,
  },
  value: {
    color: "#fff",
    fontSize: 18,
    fontWeight: "700",
  },
  valueSmall: {
    color: "#b6c2d6",
    fontSize: 13,
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
  secondaryButton: {
    marginTop: 14,
    backgroundColor: "#19253d",
    paddingVertical: 14,
    alignItems: "center",
    borderRadius: 14,
  },
  buttonText: {
    color: "#fff",
    fontWeight: "700",
  },
  status: {
    color: "#9db1d1",
    marginTop: 12,
  },
});
