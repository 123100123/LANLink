import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import { CameraView, useCameraPermissions } from "expo-camera";

import { checkHealth, pairDevice } from "@/lib/api/http";
import { loadPreferences, savePreferences } from "@/lib/storage/preferences";
import { useSessionStore } from "@/store/sessionStore";

type PairPayload = {
  t?: string;
  type?: string;
  v?: number;
  version?: number;
  a?: string;
  address?: string;
  addresses?: string[];
  tk?: string;
  token?: string;
};

function parsePairQR(data: string): { address: string; token: string } | null {
  try {
    const json = JSON.parse(data) as PairPayload;
    if (json.t !== "l" && json.type !== "lanlink_pair") return null;

    const token = json.tk ?? json.token;
    if (!token) return null;

    const address = json.a ?? json.address ?? json.addresses?.[0];
    if (!address) return null;

    return { address, token };
  } catch {
    return null;
  }
}

export default function PairScreen() {
  const router = useRouter();
  const hydrate = useSessionStore((state) => state.hydrate);
  const hydrated = useSessionStore((state) => state.hydrated);
  const hasCredentials = useSessionStore((state) => state.hasCredentials);
  const credentials = useSessionStore((state) => state.credentials);
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const setAgentAddress = useSessionStore((state) => state.setAgentAddress);
  const setCredentials = useSessionStore((state) => state.setCredentials);

  const [deviceName, setDeviceName] = useState("lanlink-mobile");
  const [token, setToken] = useState("123456");
  const [address, setAddress] = useState(agentAddress);
  const [status, setStatus] = useState<string>("Ready");
  const [loading, setLoading] = useState(false);
  const [healthLoading, setHealthLoading] = useState(false);
  const [scanning, setScanning] = useState(false);

  const [permission, requestPermission] = useCameraPermissions();

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  useEffect(() => {
    if (hydrated && hasCredentials && credentials?.deviceId) {
      router.replace("/(tabs)/device");
    }
  }, [hydrated, hasCredentials, credentials?.deviceId, router]);

  useEffect(() => {
    setAddress(agentAddress);
  }, [agentAddress]);

  async function persistAddress(addr: string) {
    setAgentAddress(addr);
    try {
      const prefs = await loadPreferences();
      await savePreferences({ ...prefs, agentAddress: addr });
    } catch {}
  }

  async function handleHealthCheck() {
    const addr = address.trim();
    if (!addr) {
      setStatus("Enter an agent address first");
      return;
    }
    setHealthLoading(true);
    setStatus("Checking...");
    try {
      const result = await checkHealth(addr);
      setStatus(`Online: ${result.status} / ${result.service}`);
      await persistAddress(addr);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Health check failed");
    } finally {
      setHealthLoading(false);
    }
  }

  async function handlePair() {
    const addr = address.trim();
    if (!addr) {
      setStatus("Enter an agent address first");
      return;
    }
    setLoading(true);
    setStatus("Pairing...");
    try {
      const result = await pairDevice(addr, {
        device_name: deviceName.trim(),
        token: token.trim(),
      });

      if (!result.device_id || !result.auth_token) {
        throw new Error("Pairing response was missing credentials");
      }

      const credentials = {
        agentAddress: addr,
        deviceId: result.device_id,
        authToken: result.auth_token,
      };

      await setCredentials(credentials);
      try {
        await savePreferences({
          agentAddress: addr,
          deviceName: deviceName.trim(),
          autoConnect: true,
        });
      } catch {}
      setStatus("Paired successfully");
      router.replace("/(tabs)/device");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Pairing failed");
    } finally {
      setLoading(false);
    }
  }

  async function handleScanQR() {
    if (!permission?.granted) {
      const result = await requestPermission();
      if (!result.granted) {
        setStatus("Camera permission denied");
        return;
      }
    }
    setScanning(true);
    setStatus("Point camera at agent QR code...");
  }

  function handleBarcodeScanned(scanned: { data: string }) {
    setScanning(false);
    const parsed = parsePairQR(scanned.data);
    if (!parsed) {
      setStatus("Invalid QR code");
      return;
    }
    setAddress(parsed.address);
    setToken(parsed.token);
    persistAddress(parsed.address);
    setStatus(`Found: ${parsed.address}`);
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Pair device</Text>
      <Text style={styles.subtitle}>
        Scan the agent QR code or enter details manually.
      </Text>

      {scanning && (
        <View style={styles.scannerContainer}>
          <CameraView
            style={styles.scanner}
            barcodeScannerSettings={{ barcodeTypes: ["qr"] }}
            onBarcodeScanned={handleBarcodeScanned}
          />
          <Pressable
            style={styles.cancelScanButton}
            onPress={() => setScanning(false)}
          >
            <Text style={styles.cancelScanText}>Cancel scan</Text>
          </Pressable>
        </View>
      )}

      <View style={styles.card}>
        <Pressable style={styles.scanButton} onPress={handleScanQR}>
          <Text style={styles.buttonText}>Scan agent QR code</Text>
        </Pressable>

        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>or enter manually</Text>
          <View style={styles.dividerLine} />
        </View>

        <Text style={styles.label}>Agent address</Text>
        <TextInput
          value={address}
          onChangeText={setAddress}
          style={styles.input}
          autoCapitalize="none"
          autoCorrect={false}
          placeholder="192.168.1.42:8787"
        />

        <Pressable
          style={styles.healthButton}
          onPress={handleHealthCheck}
          disabled={healthLoading}
        >
          {healthLoading ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Health check</Text>
          )}
        </Pressable>

        <Text style={styles.label}>Device name</Text>
        <TextInput
          value={deviceName}
          onChangeText={setDeviceName}
          style={styles.input}
          autoCapitalize="none"
        />

        <Text style={styles.label}>Pairing token</Text>
        <TextInput
          value={token}
          onChangeText={setToken}
          style={styles.input}
          secureTextEntry
        />

        <Pressable style={styles.pairButton} onPress={handlePair} disabled={loading}>
          {loading ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Pair and save</Text>
          )}
        </Pressable>

        <Text style={styles.status}>{status}</Text>
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
    marginTop: 32,
  },
  subtitle: {
    color: "#b6c2d6",
    fontSize: 15,
    marginTop: 8,
    marginBottom: 24,
  },
  scannerContainer: {
    marginBottom: 16,
    borderRadius: 16,
    overflow: "hidden",
  },
  scanner: {
    width: "100%",
    height: 300,
  },
  cancelScanButton: {
    backgroundColor: "#b94d4d",
    paddingVertical: 12,
    alignItems: "center",
  },
  cancelScanText: {
    color: "#fff",
    fontWeight: "700",
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 20,
    padding: 18,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  scanButton: {
    backgroundColor: "#4f7cff",
    paddingVertical: 14,
    alignItems: "center",
    borderRadius: 14,
  },
  divider: {
    flexDirection: "row",
    alignItems: "center",
    marginVertical: 16,
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: "#1d2a44",
  },
  dividerText: {
    color: "#7d8aa5",
    marginHorizontal: 12,
    fontSize: 13,
  },
  label: {
    color: "#d9e2f2",
    fontWeight: "600",
    marginBottom: 8,
    marginTop: 10,
  },
  input: {
    backgroundColor: "#09101d",
    borderRadius: 14,
    paddingHorizontal: 14,
    paddingVertical: 12,
    color: "#fff",
    borderWidth: 1,
    borderColor: "#23324f",
  },
  healthButton: {
    marginTop: 12,
    backgroundColor: "#19253d",
    paddingVertical: 12,
    alignItems: "center",
    borderRadius: 14,
  },
  pairButton: {
    marginTop: 18,
    backgroundColor: "#4f7cff",
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
