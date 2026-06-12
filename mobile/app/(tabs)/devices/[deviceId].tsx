import * as DocumentPicker from "expo-document-picker";
import { useLocalSearchParams } from "expo-router";
import { useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";

import { useDevicesQuery } from "@/hooks/useDevices";
import { usePing } from "@/hooks/usePing";
import { useSocket } from "@/hooks/useSocket";
import { useSessionStore } from "@/store/sessionStore";
import { uploadFile } from "@/lib/transfer/uploader";

export default function DeviceDetailScreen() {
  const { deviceId } = useLocalSearchParams<{ deviceId: string }>();
  const devices = useDevicesQuery();
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const credentials = useSessionStore((state) => state.credentials);
  const socket = useSocket();
  const ping = usePing();
  const device = devices.data?.devices.find((item) => item.device_id === deviceId);

  const [message, setMessage] = useState("");
  const [sendStatus, setSendStatus] = useState("");
  const [fileStatus, setFileStatus] = useState("");
  const [busy, setBusy] = useState(false);

  async function connectIfNeeded() {
    await socket.ensureConnected();
  }

  async function handleSendMessage() {
    setBusy(true);
    setSendStatus("Sending...");
    try {
      await connectIfNeeded();
      const response = await socket.sendDirectMessage(message);
      setSendStatus(`Agent replied: ${response.status}`);
      setMessage("");
    } catch (error) {
      setSendStatus(error instanceof Error ? error.message : "Failed to send message");
    } finally {
      setBusy(false);
    }
  }

  async function handlePing() {
    setBusy(true);
    try {
      await connectIfNeeded();
      const result = await ping.runPing(deviceId);
      ping.setResult(result);
    } catch (error) {
      ping.setError(error instanceof Error ? error.message : "Ping failed");
    } finally {
      setBusy(false);
    }
  }

  async function handlePickAndUpload() {
    setBusy(true);
    setFileStatus("Picking file...");
    try {
      const result = await DocumentPicker.getDocumentAsync({ copyToCacheDirectory: true, multiple: false });
      if (result.canceled) {
        setFileStatus("File selection canceled");
        return;
      }

      const file = result.assets[0];
      await connectIfNeeded();
      const response = await uploadFile(socket, file);
      setFileStatus(`Saved as ${response.path ?? "unknown path"}`);
    } catch (error) {
      setFileStatus(error instanceof Error ? error.message : "Upload failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>{device?.device_name ?? "Device"}</Text>
      <Text style={styles.subtitle}>{deviceId}</Text>
      <Text style={styles.subtitle}>{agentAddress || "No agent configured"}</Text>

      <View style={styles.card}>
        <Text style={styles.section}>Ping</Text>
        <Text style={styles.body}>
          {ping.status ? `${ping.status}${typeof ping.latencyMs === "number" ? ` - ${ping.latencyMs.toFixed(1)} ms` : ""}` : "Run a latency test over the authenticated socket."}
        </Text>
        <Pressable style={styles.button} onPress={handlePing} disabled={busy}>
          {busy ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Ping device</Text>}
        </Pressable>
      </View>

      <View style={styles.card}>
        <Text style={styles.section}>Direct message</Text>
        <TextInput
          value={message}
          onChangeText={setMessage}
          placeholder="Write a message..."
          placeholderTextColor="#5f6f8f"
          style={styles.input}
          multiline
        />
        <Pressable style={styles.button} onPress={handleSendMessage} disabled={busy || !message.trim()}>
          <Text style={styles.buttonText}>Send message</Text>
        </Pressable>
        <Text style={styles.status}>{sendStatus}</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.section}>File upload</Text>
        <Text style={styles.body}>Pick a file and send it through the chunked WebSocket transfer flow.</Text>
        <Pressable style={styles.secondaryButton} onPress={handlePickAndUpload} disabled={busy}>
          <Text style={styles.secondaryButtonText}>Pick file and upload</Text>
        </Pressable>
        <Text style={styles.status}>{fileStatus}</Text>
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
  },
  subtitle: {
    color: "#b6c2d6",
    marginTop: 8,
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 20,
    padding: 18,
    marginTop: 16,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  section: {
    color: "#fff",
    fontSize: 18,
    fontWeight: "700",
    marginBottom: 10,
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
  input: {
    minHeight: 96,
    backgroundColor: "#09101d",
    borderRadius: 14,
    paddingHorizontal: 14,
    paddingVertical: 12,
    color: "#fff",
    borderWidth: 1,
    borderColor: "#23324f",
    textAlignVertical: "top",
  },
  status: {
    color: "#9db1d1",
    marginTop: 12,
  },
});
