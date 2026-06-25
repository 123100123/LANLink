import * as DocumentPicker from "expo-document-picker";
import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import {
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";

import { checkHealth } from "@/lib/api/http";
import { enqueueFiles } from "@/lib/transfer/transferManager";
import { useSessionStore } from "@/store/sessionStore";
import { useTransferStore } from "@/store/transferStore";

type Reach = "checking" | "online" | "offline";

function formatBytes(bytes: number): string {
  if (bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let v = bytes;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export default function SendScreen() {
  const router = useRouter();
  const credentials = useSessionStore((s) => s.credentials);
  const agentAddress = useSessionStore((s) => s.agentAddress);
  const transfers = useTransferStore((s) => s.transfers);

  const [reach, setReach] = useState<Reach>("checking");
  const [pickStatus, setPickStatus] = useState("");

  useEffect(() => {
    if (!agentAddress) return;
    let alive = true;
    setReach("checking");
    checkHealth(agentAddress)
      .then(() => alive && setReach("online"))
      .catch(() => alive && setReach("offline"));
    return () => {
      alive = false;
    };
  }, [agentAddress]);

  const active = transfers.filter(
    (t) => t.status === "uploading" || t.status === "waiting",
  );
  const doneCount = transfers.filter((t) => t.status === "completed").length;

  async function handleSendFile() {
    if (!agentAddress || !credentials?.authToken) {
      setPickStatus("Not paired yet");
      return;
    }
    try {
      const result = await DocumentPicker.getDocumentAsync({
        // Don't copy into the cache — that blocks the UI on large files. The
        // uploader streams the content:// URI directly.
        copyToCacheDirectory: false,
        multiple: true,
      });
      if (result.canceled || result.assets.length === 0) return;

      const count = enqueueFiles(
        result.assets.map((a) => ({
          uri: a.uri,
          name: a.name ?? "unknown file",
          size: a.size ?? 0,
          mimeType: a.mimeType,
        })),
        agentAddress,
        credentials.authToken,
      );
      setPickStatus(`${count} file${count > 1 ? "s" : ""} added — sending…`);
    } catch (error) {
      setPickStatus(error instanceof Error ? error.message : "Failed to pick files");
    }
  }

  const reachColor =
    reach === "online" ? "#5ed39b" : reach === "offline" ? "#ff7b7b" : "#7d8aa5";
  const reachLabel =
    reach === "online" ? "Online" : reach === "offline" ? "Unreachable" : "Checking…";

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Send</Text>

      <View style={styles.statusCard}>
        <View style={[styles.dot, { backgroundColor: reachColor }]} />
        <View style={{ flex: 1 }}>
          <Text style={styles.statusLabel}>Connected to</Text>
          <Text style={styles.statusValue} numberOfLines={1}>
            {agentAddress || "—"}
          </Text>
        </View>
        <Text style={[styles.statusBadge, { color: reachColor }]}>{reachLabel}</Text>
      </View>

      <Pressable style={styles.primaryButton} onPress={handleSendFile}>
        <Text style={styles.primaryButtonText}>＋  Select files to send</Text>
      </Pressable>
      {pickStatus ? <Text style={styles.pickStatus}>{pickStatus}</Text> : null}

      {active.length > 0 && (
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Sending now</Text>
          {active.map((t) => {
            const pct = Math.round(t.progress * 100);
            return (
              <View key={t.id} style={styles.transferRow}>
                <View style={styles.transferTop}>
                  <Text style={styles.transferName} numberOfLines={1}>
                    {t.filename}
                  </Text>
                  <Text style={styles.transferPct}>
                    {t.status === "waiting" ? "Waiting" : `${pct}%`}
                  </Text>
                </View>
                <View style={styles.progressTrack}>
                  <View
                    style={[styles.progressFill, { width: `${Math.max(2, pct)}%` }]}
                  />
                </View>
                <Text style={styles.transferMeta}>
                  {formatBytes(t.sentBytes)} / {formatBytes(t.size)}
                </Text>
              </View>
            );
          })}
        </View>
      )}

      <Pressable
        style={styles.linkButton}
        onPress={() => router.push("/(tabs)/transfers")}
      >
        <Text style={styles.linkText}>
          View all transfers{doneCount > 0 ? `  ·  ${doneCount} done` : ""}  →
        </Text>
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
    fontSize: 30,
    fontWeight: "800",
    marginTop: 24,
    marginBottom: 16,
  },
  statusCard: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    backgroundColor: "#121b2f",
    borderRadius: 16,
    padding: 16,
    borderWidth: 1,
    borderColor: "#1d2a44",
    marginBottom: 20,
  },
  dot: {
    width: 10,
    height: 10,
    borderRadius: 5,
  },
  statusLabel: {
    color: "#7d8aa5",
    fontSize: 12,
  },
  statusValue: {
    color: "#fff",
    fontSize: 15,
    fontWeight: "700",
    marginTop: 2,
  },
  statusBadge: {
    fontSize: 12,
    fontWeight: "700",
  },
  primaryButton: {
    backgroundColor: "#4f7cff",
    paddingVertical: 18,
    borderRadius: 16,
    alignItems: "center",
  },
  primaryButtonText: {
    color: "#fff",
    fontWeight: "800",
    fontSize: 16,
  },
  pickStatus: {
    color: "#9db1d1",
    marginTop: 12,
    textAlign: "center",
  },
  section: {
    marginTop: 24,
  },
  sectionTitle: {
    color: "#d9e2f2",
    fontWeight: "700",
    fontSize: 13,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 10,
  },
  transferRow: {
    backgroundColor: "#121b2f",
    borderRadius: 14,
    padding: 14,
    marginBottom: 10,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  transferTop: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    gap: 12,
  },
  transferName: {
    color: "#fff",
    fontWeight: "600",
    flex: 1,
  },
  transferPct: {
    color: "#4f7cff",
    fontWeight: "700",
  },
  progressTrack: {
    height: 8,
    backgroundColor: "#09101d",
    borderRadius: 999,
    overflow: "hidden",
    marginTop: 10,
  },
  progressFill: {
    height: "100%",
    backgroundColor: "#4f7cff",
    borderRadius: 999,
  },
  transferMeta: {
    color: "#9db1d1",
    fontSize: 12,
    marginTop: 8,
  },
  linkButton: {
    marginTop: 22,
    paddingVertical: 12,
    alignItems: "center",
  },
  linkText: {
    color: "#7d8aa5",
    fontWeight: "600",
  },
});
