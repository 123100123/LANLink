import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { Pressable, ScrollView, StyleSheet, Switch, Text, TextInput, View } from "react-native";

import { clearCredentials } from "@/lib/storage/credentials";
import { loadPreferences, savePreferences } from "@/lib/storage/preferences";
import { useSessionStore } from "@/store/sessionStore";

export default function SettingsScreen() {
  const router = useRouter();
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const credentials = useSessionStore((state) => state.credentials);
  const setAgentAddress = useSessionStore((state) => state.setAgentAddress);
  const clearSession = useSessionStore((state) => state.clearSession);
  const [deviceName, setDeviceName] = useState("lanlink-mobile");
  const [autoConnect, setAutoConnect] = useState(true);
  const [address, setAddress] = useState(agentAddress);
  const [status, setStatus] = useState("");

  useEffect(() => {
    setAddress(agentAddress);
  }, [agentAddress]);

  useEffect(() => {
    loadPreferences().then((prefs) => {
      setAutoConnect(prefs.autoConnect ?? true);
      if (prefs.deviceName) {
        setDeviceName(prefs.deviceName);
      }
    });
  }, []);

  async function handleSave() {
    await savePreferences({
      agentAddress: address.trim(),
      deviceName: deviceName.trim(),
      autoConnect,
    });
    setAgentAddress(address.trim());
    setStatus("Settings saved");
  }

  async function handleClearCredentials() {
    await clearCredentials();
    clearSession();
    router.replace("/pair");
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Settings</Text>

      <View style={styles.card}>
        <Text style={styles.label}>Agent address</Text>
        <TextInput value={address} onChangeText={setAddress} style={styles.input} autoCapitalize="none" />

        <Text style={styles.label}>Device name</Text>
        <TextInput value={deviceName} onChangeText={setDeviceName} style={styles.input} autoCapitalize="none" />

        <View style={styles.switchRow}>
          <Text style={styles.label}>Auto connect</Text>
          <Switch value={autoConnect} onValueChange={setAutoConnect} />
        </View>

        <Pressable style={styles.button} onPress={handleSave}>
          <Text style={styles.buttonText}>Save settings</Text>
        </Pressable>

        <Text style={styles.status}>{status}</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.label}>Session</Text>
        <Text style={styles.body}>{credentials ? `Paired as ${credentials.deviceId}` : "No credentials saved"}</Text>
        <Pressable style={styles.dangerButton} onPress={handleClearCredentials}>
          <Text style={styles.buttonText}>Clear credentials</Text>
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
    marginBottom: 8,
    marginTop: 10,
  },
  body: {
    color: "#b6c2d6",
    lineHeight: 20,
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
  dangerButton: {
    marginTop: 18,
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
    marginTop: 12,
  },
  switchRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginTop: 10,
  },
});
