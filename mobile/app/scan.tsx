import { useFocusEffect, useRouter } from "expo-router";
import { useCallback, useRef, useState } from "react";
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

const RESCAN_DELAY_MS = 3000;

export default function ScanScreen() {
  const router = useRouter();
  const setCredentials = useSessionStore((state) => state.setCredentials);

  const [port, setPort] = useState("8787");
  const [scanning, setScanning] = useState(false);
  const [hosts, setHosts] = useState<DiscoveredHost[]>([]);
  const [status, setStatus] = useState("Searching your network…");
  const [connecting, setConnecting] = useState<string | null>(null);

  const activeRef = useRef(false);
  const connectingRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);
  const baseRef = useRef<string | null>(null);
  const portRef = useRef(port);
  portRef.current = port;

  const delay = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

  // Live discovery: scan automatically while the screen is focused, refreshing
  // the list every few seconds. No manual tap needed.
  useFocusEffect(
    useCallback(() => {
      activeRef.current = true;
      connectingRef.current = false;
      baseRef.current = null;
      setHosts([]);
      setConnecting(null);
      runLoop();
      return () => {
        activeRef.current = false;
        abortRef.current?.abort();
      };
    }, []),
  );

  async function runLoop() {
    while (activeRef.current) {
      await runScan();
      if (!activeRef.current) break;
      await delay(RESCAN_DELAY_MS);
    }
  }

  async function runScan() {
    if (connectingRef.current) return;

    const portNum = parseInt(portRef.current.trim(), 10) || 8787;

    if (!baseRef.current) {
      const base = await getSubnetBase();
      if (!base) {
        if (activeRef.current) setStatus("Connect to Wi-Fi to scan.");
        return;
      }
      baseRef.current = base;
    }
    if (!activeRef.current) return;

    setScanning(true);
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const found = await sweepSubnet(baseRef.current, portNum, undefined, controller.signal);
      if (!activeRef.current) return;
      setHosts(found);
      setStatus(
        found.length > 0
          ? `${found.length} receiver${found.length > 1 ? "s" : ""} found — tap to connect`
          : "No receivers found yet — still searching…",
      );
    } catch {
      // ignore transient errors; the next pass retries
    } finally {
      if (activeRef.current) setScanning(false);
    }
  }

  async function handleConnect(host: DiscoveredHost) {
    connectingRef.current = true;
    activeRef.current = false;
    abortRef.current?.abort();
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
      connectingRef.current = false;
      // resume live scanning
      activeRef.current = true;
      runLoop();
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Scan network</Text>
      <Text style={styles.subtitle}>
        Receivers on your Wi-Fi appear automatically. Tap one to connect — no token needed.
      </Text>

      <View style={styles.card}>
        <View style={styles.statusRow}>
          {scanning && <ActivityIndicator color="#4f7cff" style={{ marginRight: 10 }} />}
          <Text style={styles.status}>{status}</Text>
        </View>

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

        <View style={styles.portRow}>
          <Text style={styles.portLabel}>Receiver port</Text>
          <TextInput
            value={port}
            onChangeText={setPort}
            style={styles.portInput}
            keyboardType="number-pad"
            placeholder="8787"
            placeholderTextColor="#5f6f8f"
          />
        </View>
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
  statusRow: {
    flexDirection: "row",
    alignItems: "center",
    marginBottom: 4,
  },
  status: {
    color: "#9db1d1",
    flex: 1,
  },
  hostRow: {
    flexDirection: "row",
    alignItems: "center",
    backgroundColor: "#09101d",
    borderRadius: 14,
    padding: 14,
    marginTop: 12,
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
  portRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginTop: 18,
    paddingTop: 14,
    borderTopWidth: 1,
    borderTopColor: "#1d2a44",
  },
  portLabel: {
    color: "#7d8aa5",
    fontSize: 13,
  },
  portInput: {
    backgroundColor: "#09101d",
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 8,
    color: "#fff",
    borderWidth: 1,
    borderColor: "#23324f",
    minWidth: 96,
    textAlign: "center",
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
