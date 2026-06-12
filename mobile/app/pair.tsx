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

import { pairDevice } from "@/lib/api/http";
import { savePreferences } from "@/lib/storage/preferences";
import { useSessionStore } from "@/store/sessionStore";

export default function PairScreen() {
  const router = useRouter();
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const setCredentials = useSessionStore((state) => state.setCredentials);

  const [deviceName, setDeviceName] = useState("lanlink-mobile");
  const [token, setToken] = useState("123456");
  const [address, setAddress] = useState(agentAddress);
  const [status, setStatus] = useState<string>("Ready");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setAddress(agentAddress);
  }, [agentAddress]);

  async function handlePair() {
    setLoading(true);
    setStatus("Pairing...");
    try {
      const result = await pairDevice(address.trim(), {
        device_name: deviceName.trim(),
        token: token.trim(),
      });

      if (!result.device_id || !result.auth_token) {
        throw new Error("Pairing response was missing credentials");
      }

      const credentials = {
        agentAddress: address.trim(),
        deviceId: result.device_id,
        authToken: result.auth_token,
      };

      await setCredentials(credentials);
      try {
        await savePreferences({
          agentAddress: address.trim(),
          deviceName: deviceName.trim(),
          autoConnect: true,
        });
      } catch {
        // Credentials are already saved; preference persistence is best-effort.
      }
      setStatus("Paired successfully");
      router.replace(`/(tabs)/devices/${result.device_id}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Pairing failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Pair device</Text>
      <Text style={styles.subtitle}>Use the shared pairing token from the agent.</Text>

      <View style={styles.card}>
        <Text style={styles.label}>Agent address</Text>
        <TextInput value={address} onChangeText={setAddress} style={styles.input} autoCapitalize="none" />

        <Text style={styles.label}>Device name</Text>
        <TextInput value={deviceName} onChangeText={setDeviceName} style={styles.input} autoCapitalize="none" />

        <Text style={styles.label}>Pairing token</Text>
        <TextInput value={token} onChangeText={setToken} style={styles.input} secureTextEntry />

        <Pressable style={styles.button} onPress={handlePair} disabled={loading}>
          {loading ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Pair and save</Text>}
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
  button: {
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
