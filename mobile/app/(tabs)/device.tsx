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
import { useTransferStore } from "@/store/transferStore";
import {
  httpTransfer,
  cancelTransfer,
} from "@/lib/transfer/httpTransfer";
import { createId } from "@/lib/protocol/envelope";

export default function DeviceScreen() {
  const credentials = useSessionStore((state) => state.credentials);
  const agentAddress = useSessionStore((state) => state.agentAddress);
  const devices = useDevicesQuery();
  const socket = useSocket();
  const ping = usePing();
  const addTransfer = useTransferStore((state) => state.addTransfer);
  const updateTransfer = useTransferStore((state) => state.updateTransfer);

  const [busy, setBusy] = useState(false);
  const [pingStatus, setPingStatus] = useState("");
  const [uploadStatus, setUploadStatus] = useState("");
  const [uploading, setUploading] = useState(false);

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

  async function handlePickAndUpload() {
    if (uploading) {
      await cancelTransfer();
      setUploading(false);
      setBusy(false);
      setUploadStatus("Cancelled");
      return;
    }

    setBusy(true);
    setUploadStatus("Picking file...");

    let transferId = "";

    try {
      const result = await DocumentPicker.getDocumentAsync({
        copyToCacheDirectory: true,
        multiple: false,
      });

      if (result.canceled) {
        setUploadStatus("");
        setBusy(false);
        return;
      }

      const file = result.assets[0];
      transferId = createId("transfer");
      setUploading(true);

      addTransfer({
        id: transferId,
        filename: file.name ?? "unknown file",
        size: file.size ?? 0,
        sentBytes: 0,
        progress: 0,
        status: "uploading",
        speed: 0,
        elapsed: 0,
        startedAt: Date.now(),
      });

      const response = await httpTransfer(
        agentAddress,
        credentials!.authToken,
        {
          uri: file.uri,
          name: file.name ?? "unknown file",
          size: file.size,
        },
        {
          transferId,
          onProgress: ({ sentBytes, totalBytes, progress, speed, elapsed }) => {
            updateTransfer(transferId, {
              sentBytes,
              size: totalBytes,
              progress,
              speed,
              elapsed,
              status: "uploading",
            });
            setUploadStatus(
              `${Math.round(progress * 100)}% · ${(speed / 1024 / 1024).toFixed(1)} MB/s`
            );
          },
        }
      );

      updateTransfer(transferId, {
        sentBytes: file.size ?? 0,
        progress: 1,
        status: "completed",
        completedAt: Date.now(),
        savedPath: response.path,
      });

      setUploadStatus(`Saved as ${response.path}`);
    } catch (error) {
      const msg = error instanceof Error ? error.message : "Upload failed";
      const isCancelled = msg === "Upload cancelled";

      if (transferId) {
        updateTransfer(transferId, {
          status: isCancelled ? "cancelled" : "failed",
          error: isCancelled ? undefined : msg,
          completedAt: Date.now(),
        });
      }

      setUploadStatus(isCancelled ? "Cancelled" : msg);
    } finally {
      setUploading(false);
      setBusy(false);
    }
  }

  const isUploading = uploading;

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
        <Text style={styles.section}>Upload file</Text>
        <Text style={styles.body}>
          {isUploading
            ? "Tap cancel to abort the upload."
            : "Pick a file to send via parallel HTTP chunks."}
        </Text>
        <Pressable
          style={isUploading ? styles.cancelButton : styles.secondaryButton}
          onPress={handlePickAndUpload}
        >
          <Text style={styles.buttonText}>
            {isUploading ? "Cancel upload" : "Pick file"}
          </Text>
        </Pressable>
        {uploadStatus ? (
          <Text style={styles.status}>{uploadStatus}</Text>
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
  cancelButton: {
    marginTop: 14,
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
});
