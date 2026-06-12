import { Link, useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";

import { checkHealth } from "@/lib/api/http";
import { loadPreferences, savePreferences } from "@/lib/storage/preferences";
import { useSessionStore } from "@/store/sessionStore";

export default function SetupScreen() {
  const router = useRouter();
  const hydrate = useSessionStore((state) => state.hydrate);
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const setAgentAddress = useSessionStore((state) => state.setAgentAddress);
  const hydrated = useSessionStore((state) => state.hydrated);
  const hasCredentials = useSessionStore((state) => state.hasCredentials);

  const [address, setAddress] = useState(agentAddress);
  const [status, setStatus] = useState<string>("Idle");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  useEffect(() => {
    setAddress(agentAddress);
  }, [agentAddress]);

  useEffect(() => {
    if (hydrated && hasCredentials) {
      router.replace("/(tabs)/home");
    }
  }, [hydrated, hasCredentials, router]);

  const normalizedAddress = useMemo(() => address.trim(), [address]);

  async function handleHealthCheck() {
    setLoading(true);
    setStatus("Checking...");
    try {
      const result = await checkHealth(normalizedAddress);
      setStatus(`Online: ${result.status} / ${result.service}`);
      setAgentAddress(normalizedAddress);
      try {
        const prefs = await loadPreferences();
        await savePreferences({
          ...prefs,
          agentAddress: normalizedAddress,
        });
      } catch {
        // Keep the health check result even if preference persistence fails.
      }
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Health check failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>LANLink</Text>
      <Text style={styles.subtitle}>Connect to a local agent on your network.</Text>

      <View style={styles.card}>
        <Text style={styles.label}>Agent address</Text>
        <TextInput
          value={address}
          onChangeText={setAddress}
          autoCapitalize="none"
          autoCorrect={false}
          placeholder="192.168.1.42:8787"
          style={styles.input}
        />

        <Pressable style={styles.button} onPress={handleHealthCheck} disabled={loading}>
          {loading ? <ActivityIndicator color="#fff" /> : <Text style={styles.buttonText}>Health check</Text>}
        </Pressable>

        <Text style={styles.status}>{status}</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.sectionTitle}>Next step</Text>
        <Text style={styles.body}>
          If the agent is reachable, go to pairing to store your credentials locally.
        </Text>
        <Link href="/pair" asChild>
          <Pressable style={styles.secondaryButton}>
            <Text style={styles.secondaryButtonText}>Go to pairing</Text>
          </Pressable>
        </Link>
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
    fontSize: 36,
    fontWeight: "800",
    marginTop: 32,
  },
  subtitle: {
    color: "#b6c2d6",
    fontSize: 16,
    marginTop: 8,
    marginBottom: 24,
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
  status: {
    color: "#9db1d1",
    marginTop: 12,
  },
  sectionTitle: {
    color: "#fff",
    fontSize: 18,
    fontWeight: "700",
    marginBottom: 8,
  },
  body: {
    color: "#b6c2d6",
    lineHeight: 20,
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
});
