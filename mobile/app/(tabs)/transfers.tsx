import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";

import { useTransferStore, type TransferItem } from "@/store/transferStore";
import {
  cancelTransfer,
  retryTransfer,
  removeTransfer,
  stopAll,
  startAll,
  clearCompleted,
} from "@/lib/transfer/transferManager";

function formatTime(timestamp?: number) {
  if (!timestamp) return "";
  return new Date(timestamp).toLocaleString();
}

function formatBytes(bytes: number) {
  if (bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatSpeed(bytesPerSec: number) {
  if (bytesPerSec <= 0) return "";
  const mbps = bytesPerSec / 1024 / 1024;
  return `${mbps.toFixed(1)} MB/s`;
}

function formatETA(sentBytes: number, totalBytes: number, speed: number) {
  if (speed <= 0 || totalBytes <= 0) return "";
  const remaining = totalBytes - sentBytes;
  const seconds = remaining / speed;
  if (seconds < 1) return "<1s";
  if (seconds < 60) return `${Math.round(seconds)}s`;
  const minutes = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  return `${minutes}m${secs}s`;
}

// The same button palette as the pair and settings screens, so actions read
// consistently across the app: blue = go, red = stop, navy = neutral.
const BUTTON_VARIANTS = {
  primary: "#4f7cff",
  danger: "#b94d4d",
  neutral: "#19253d",
} as const;

type ButtonVariant = keyof typeof BUTTON_VARIANTS;

function ActionButton({
  label,
  variant,
  onPress,
}: {
  label: string;
  variant: ButtonVariant;
  onPress: () => void;
}) {
  return (
    <Pressable
      style={[styles.actionButton, { backgroundColor: BUTTON_VARIANTS[variant] }]}
      onPress={onPress}
    >
      <Text style={styles.actionText}>{label}</Text>
    </Pressable>
  );
}

function TransferCard({ transfer }: { transfer: TransferItem }) {
  const percent = Math.round(transfer.progress * 100);

  return (
    <View style={styles.card}>
      <View style={styles.transferHeader}>
        <Text style={styles.filename} numberOfLines={1}>
          {transfer.filename}
        </Text>
        <Text style={[styles.status, statusColor(transfer.status)]}>
          {statusLabel(transfer, percent)}
        </Text>
      </View>

      {transfer.status === "uploading" && (
        <>
          <View style={styles.progressTrack}>
            <View
              style={[
                styles.progressFill,
                { width: `${Math.max(2, percent)}%` },
              ]}
            />
          </View>
          <View style={styles.statsRow}>
            <Text style={styles.meta}>
              {formatBytes(transfer.sentBytes)} / {formatBytes(transfer.size)}
            </Text>
            {transfer.speed > 0 && (
              <Text style={styles.meta}>{formatSpeed(transfer.speed)}</Text>
            )}
          </View>
          {transfer.speed > 0 && (
            <Text style={styles.meta}>
              ETA{" "}
              {formatETA(transfer.sentBytes, transfer.size, transfer.speed)}
            </Text>
          )}
          <View style={styles.actions}>
            <ActionButton
              label="Cancel"
              variant="danger"
              onPress={() => cancelTransfer(transfer.id)}
            />
          </View>
        </>
      )}

      {transfer.status === "waiting" && (
        <View style={styles.actions}>
          <ActionButton
            label="Cancel"
            variant="danger"
            onPress={() => cancelTransfer(transfer.id)}
          />
        </View>
      )}

      {transfer.status === "completed" && (
        <>
          <Text style={styles.meta}>
            {formatBytes(transfer.size)} sent
            {transfer.speed > 0 && ` at ${formatSpeed(transfer.speed)}`}
          </Text>
          <Text style={styles.meta}>
            Finished {formatTime(transfer.completedAt)}
          </Text>
          {transfer.savedPath && (
            <Text style={styles.path}>{transfer.savedPath}</Text>
          )}
          <View style={styles.actions}>
            <ActionButton
              label="Remove"
              variant="neutral"
              onPress={() => removeTransfer(transfer.id)}
            />
          </View>
        </>
      )}

      {transfer.status === "failed" && (
        <>
          <Text style={styles.error}>
            {transfer.error ?? "Transfer failed"}
          </Text>
          <View style={styles.actions}>
            <ActionButton
              label="Retry"
              variant="primary"
              onPress={() => retryTransfer(transfer.id)}
            />
            <ActionButton
              label="Remove"
              variant="neutral"
              onPress={() => removeTransfer(transfer.id)}
            />
          </View>
        </>
      )}

      {transfer.status === "cancelled" && (
        <>
          <Text style={styles.meta}>Cancelled</Text>
          <View style={styles.actions}>
            <ActionButton
              label="Retry"
              variant="primary"
              onPress={() => retryTransfer(transfer.id)}
            />
            <ActionButton
              label="Remove"
              variant="neutral"
              onPress={() => removeTransfer(transfer.id)}
            />
          </View>
        </>
      )}
    </View>
  );
}

function statusLabel(t: TransferItem, percent: number) {
  switch (t.status) {
    case "waiting":
      return "Waiting";
    case "uploading":
      return `${percent}%`;
    case "completed":
      return "Done";
    case "failed":
      return "Failed";
    case "cancelled":
      return "Cancelled";
  }
}

function statusColor(status: TransferItem["status"]) {
  switch (status) {
    case "completed":
      return { color: "#6fcf97" };
    case "failed":
      return { color: "#ff8a8a" };
    case "cancelled":
      return { color: "#f0c674" };
    case "waiting":
      return { color: "#9db1d1" };
    // "uploading" — the live percentage. Without an explicit color it inherited
    // the default black, which is invisible on the dark card.
    default:
      return { color: "#4f7cff" };
  }
}

export default function TransfersScreen() {
  const transfers = useTransferStore((s) => s.transfers);

  const hasTransfers = transfers.length > 0;
  const hasCompleted = transfers.some((t) => t.status === "completed");
  const hasFailedOrCancelled = transfers.some(
    (t) => t.status === "failed" || t.status === "cancelled"
  );
  const hasUploadingOrWaiting = transfers.some(
    (t) => t.status === "uploading" || t.status === "waiting"
  );

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <View style={styles.headerRow}>
        <View>
          <Text style={styles.title}>Transfers</Text>
          <Text style={styles.subtitle}>Upload queue</Text>
        </View>
      </View>

      {hasTransfers && (
        <View style={styles.globalActions}>
          {hasUploadingOrWaiting && (
            <ActionButton
              label="Stop all"
              variant="danger"
              onPress={() => stopAll()}
            />
          )}
          {hasFailedOrCancelled && (
            <ActionButton
              label="Start all"
              variant="primary"
              onPress={() => startAll()}
            />
          )}
          {hasCompleted && (
            <ActionButton
              label="Clear done"
              variant="neutral"
              onPress={() => clearCompleted()}
            />
          )}
        </View>
      )}

      {transfers.length === 0 ? (
        <View style={styles.card}>
          <Text style={styles.emptyTitle}>No transfers yet</Text>
          <Text style={styles.body}>
            Send files from the Device tab to see them here.
          </Text>
        </View>
      ) : (
        transfers.map((transfer) => (
          <TransferCard key={transfer.id} transfer={transfer} />
        ))
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
  globalActions: {
    flexDirection: "row",
    gap: 8,
    marginBottom: 16,
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
  statsRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    marginTop: 10,
  },
  meta: {
    color: "#9db1d1",
    marginTop: 4,
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
  actions: {
    flexDirection: "row",
    gap: 8,
    marginTop: 12,
  },
  actionButton: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 10,
  },
  actionText: {
    color: "#fff",
    fontWeight: "700",
    fontSize: 13,
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
