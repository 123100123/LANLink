import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";

import { useTransferStore } from "@/store/transferStore";

function formatTime(timestamp?: number) {
  if (!timestamp) {
    return "";
  }

  return new Date(timestamp).toLocaleString();
}

function formatBytes(bytes?: number) {
  if (!bytes || bytes <= 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${value.toFixed(unitIndex === 0 ? 0 : 2)} ${units[unitIndex]}`;
}

export default function TransfersScreen() {
  const transfers = useTransferStore((state) => state.transfers);
  const clearTransfers = useTransferStore((state) => state.clearTransfers);

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <View style={styles.headerRow}>
        <View>
          <Text style={styles.title}>Transfers</Text>
          <Text style={styles.subtitle}>Recent file uploads</Text>
        </View>

        {transfers.length > 0 ? (
          <Pressable style={styles.clearButton} onPress={clearTransfers}>
            <Text style={styles.clearButtonText}>Clear</Text>
          </Pressable>
        ) : null}
      </View>

      {transfers.length === 0 ? (
        <View style={styles.card}>
          <Text style={styles.emptyTitle}>No transfers yet</Text>
          <Text style={styles.body}>
            Sent files will appear here with progress and completion time.
          </Text>
        </View>
      ) : (
        transfers.map((transfer) => {
          const percent = Math.round(transfer.progress * 100);

          return (
            <View key={transfer.id} style={styles.card}>
              <View style={styles.transferHeader}>
                <Text style={styles.filename} numberOfLines={1}>
                  {transfer.filename}
                </Text>

                <Text style={styles.status}>
                  {transfer.status === "uploading"
                    ? `${percent}%`
                    : transfer.status === "completed"
                      ? "Sent"
                      : "Failed"}
                </Text>
              </View>

              {transfer.status === "uploading" ? (
                <>
                  <View style={styles.progressTrack}>
                    <View
                      style={[
                        styles.progressFill,
                        { width: `${Math.max(2, percent)}%` },
                      ]}
                    />
                  </View>

                  <Text style={styles.meta}>
                    {formatBytes(transfer.sentBytes)} /{" "}
                    {formatBytes(transfer.size)}
                  </Text>
                </>
              ) : null}

              {transfer.status === "completed" ? (
                <>
                  <Text style={styles.meta}>
                    Sent at {formatTime(transfer.completedAt)}
                  </Text>

                  {transfer.savedPath ? (
                    <Text style={styles.path}>{transfer.savedPath}</Text>
                  ) : null}
                </>
              ) : null}

              {transfer.status === "failed" ? (
                <Text style={styles.error}>
                  {transfer.error ?? "Transfer failed"}
                </Text>
              ) : null}
            </View>
          );
        })
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flexGrow: 1,
    padding: 20,
    backgroundColor: "#0b1220",
  },
  headerRow: {
    marginTop: 24,
    marginBottom: 16,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
  },
  title: {
    color: "#fff",
    fontSize: 30,
    fontWeight: "800",
  },
  subtitle: {
    color: "#b6c2d6",
    marginTop: 8,
  },
  clearButton: {
    backgroundColor: "#19253d",
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderRadius: 12,
  },
  clearButtonText: {
    color: "#fff",
    fontWeight: "700",
  },
  card: {
    backgroundColor: "#121b2f",
    borderRadius: 20,
    padding: 18,
    marginBottom: 14,
    borderWidth: 1,
    borderColor: "#1d2a44",
  },
  transferHeader: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
  },
  filename: {
    color: "#fff",
    fontSize: 16,
    fontWeight: "700",
    flex: 1,
  },
  status: {
    color: "#9db1d1",
    fontWeight: "700",
  },
  progressTrack: {
    height: 8,
    backgroundColor: "#09101d",
    borderRadius: 999,
    overflow: "hidden",
    marginTop: 14,
  },
  progressFill: {
    height: "100%",
    backgroundColor: "#4f7cff",
    borderRadius: 999,
  },
  meta: {
    color: "#9db1d1",
    marginTop: 10,
  },
  path: {
    color: "#6f7f9d",
    marginTop: 8,
    fontSize: 12,
  },
  error: {
    color: "#ff8a8a",
    marginTop: 10,
  },
  emptyTitle: {
    color: "#fff",
    fontSize: 18,
    fontWeight: "700",
    marginBottom: 8,
  },
  body: {
    color: "#b6c2d6",
    lineHeight: 20,
  },
});