import { useRouter } from "expo-router";
import { useRef, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";

import { pairAuto } from "@/lib/api/http";
import { getSubnetBase, sweepSubnet, type DiscoveredHost } from "@/lib/discovery/sweep";
import { savePreferences } from "@/lib/storage/preferences";
import { useSessionStore } from "@/store/sessionStore";

export default function ScanScreen() {
  const router = useRouter();
  const setCredentials = useSessionStore((state) => state.setCredentials);

  const [port, setPort] = useState("8787");
  const [scanning, setScanning] = useState(false);
  const [progress, setProgress] = useState(0);
  const [hosts, setHosts] = useState<DiscoveredHost[]>([]);
  const [status, setStatus] = useState("Scan your network to find a receiver.");
  const [connecting, setConnecting] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  async function handleScan() {
    const portNum = parseInt(port.trim(), 10) || 8787;
    setHosts([]);
    setProgress(0);
    setScanning(true);
    setStatus("Locating your network…");

    const base = await getSubnetBase();
    if (!base) {
      setScanning(false);
      setStatus("Could not determine your network. Connect to Wi-Fi and try again.");
      return;
    }

    setStatus(`Scanning ${base}0/24 on port ${portNum}…`);
    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const found = await sweepSubnet(
        base,
        portNum,
        (done, total) => setProgress(Math.round((done / total) * 100)),
        controller.signal,
      );
      setHosts(found);
      setStatus(
        found.length > 0
          ? `Found ${found.length} receiver(s). Tap one to connect.`
          : "No receivers found on this network.",
      );
    } catch {
      setStatus("Scan failed. Try again.");
    } finally {
      setScanning(false);
      abortRef.current = null;
    }
  }

  function handleCancel() {
    abortRef.current?.abort();
    setScanning(false);
    setStatus("Scan cancelled.");
  }

  async function handleConnect(host: DiscoveredHost) {
    setConnecting(host.address);
    setStatus(`Connecting to ${host.address}…`);
    try {
      const result = await pairAuto(host.address, "lanlink-mobile");
      if (!result.device_id || !result.auth_token) {
        throw new Error("Auto-connect response was missing credentials");
      }
      await setCredentials({
        agentAddress: host.address,
        deviceId: result.device_id,
        authToken: result.auth_token,
      });
      try {
        await savePreferences({
          agentAddress: host.address,
          deviceName: "lanlink-mobile",
          autoConnect: true,
        });
      } catch {}
      router.replace("/(tabs)/device");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Auto-connect failed");
      setConnecting(null);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Scan network</Text>
      <Text style={styles.subtitle}>
        Find LANLink receivers on your Wi-Fi and connect without a token.
      </Text>

      <View style={styles.card}>
        <Text style={styles.label}>Receiver port</Text>
        <TextInput
          value={port}
          onChangeText={setPort}
          style={styles.input}
          keyboardType="number-pad"
          placeholder="8787"
          placeholderTextColor="#5f6f8f"
        />

        {scanning ? (
          <>
            <Pressable style={styles.cancelButton} onPress={handleCancel}>
              <Text style={styles.buttonText}>Cancel ({progress}%)</Text>
            </Pressable>
            <ActivityIndicator color="#4f7cff" style={{ marginTop: 14 }} />
          </>
        ) : (
          <Pressable style={styles.scanButton} onPress={handleScan}>
            <Text style={styles.buttonText}>Scan network</Text>
          </Pressable>
        )}

        <Text style={styles.status}>{status}</Text>

        {hosts.map((host) => (
          <Pressable
            key={host.address}
            style={styles.hostRow}
            onPress={() => handleConnect(host)}
            disabled={connecting !== null}
          >
            <View style={{ flex: 1 }}>
              <Text style={styles.hostAddr}>{host.address}</Text>
              <Text style={styles.hostService}>{host.service}</Text>
            </View>
            {connecting === host.address ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.connectLabel}>Connect</Text>
            )}
          </Pressable>
        ))}
      </View>

      <Pressable style={styles.backButton} onPress={() => router.back()}>
        <Text style={styles.backText}>Back to manual pairing</Text>
      </Pressable>
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
    marginTop: 32,
  },
  subtitle: {
    color: "#b6c2d6",
    fontSize: 15,
    marginTop: 8,
    marginBottom: 24,
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 20,
    padding: 18,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  label: {
    color: "#d9e2f2",
    fontWeight: "600",
    marginBottom: 8,
  },
  input: {
    backgroundColor: "#09101d",
    borderRadius: 14,
    paddingHorizontal: 14,
    paddingVertical: 12,
    color: "#fff",
    borderWidth: 1,
    borderColor: "#23324f",
    marginBottom: 14,
  },
  scanButton: {
    backgroundColor: "#4f7cff",
    paddingVertical: 14,
    alignItems: "center",
    borderRadius: 14,
  },
  cancelButton: {
    backgroundColor: "#b94d4d",
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
    marginTop: 14,
    marginBottom: 4,
  },
  hostRow: {
    flexDirection: "row",
    alignItems: "center",
    backgroundColor: "#09101d",
    borderRadius: 14,
    padding: 14,
    marginTop: 10,
    borderWidth: 1,
    borderColor: "#23324f",
  },
  hostAddr: {
    color: "#fff",
    fontWeight: "700",
    fontSize: 15,
  },
  hostService: {
    color: "#7d8aa5",
    fontSize: 12,
    marginTop: 2,
  },
  connectLabel: {
    color: "#4f7cff",
    fontWeight: "700",
  },
  backButton: {
    marginTop: 20,
    paddingVertical: 12,
    alignItems: "center",
  },
  backText: {
    color: "#7d8aa5",
    fontWeight: "600",
  },
});
