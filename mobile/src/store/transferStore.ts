import { create } from "zustand";

export type TransferStatus = "queued" | "uploading" | "completed" | "failed" | "cancelled";

export type TransferItem = {
  id: string;
  filename: string;
  size: number;
  sentBytes: number;
  progress: number;
  status: TransferStatus;
  speed: number;
  elapsed: number;
  startedAt: number;
  completedAt?: number;
  error?: string;
  savedPath?: string;
};

type TransferStore = {
  transfers: TransferItem[];
  maxConcurrency: number;
  activeCount: number;
  addTransfer: (transfer: TransferItem) => void;
  updateTransfer: (id: string, patch: Partial<Omit<TransferItem, "id">>) => void;
  removeTransfer: (id: string) => void;
  clearTransfers: () => void;
  setActiveCount: (count: number) => void;
  setMaxConcurrency: (max: number) => void;
};

export const useTransferStore = create<TransferStore>((set) => ({
  transfers: [],
  maxConcurrency: 4,
  activeCount: 0,

  addTransfer: (transfer) =>
    set((state) => ({
      transfers: [transfer, ...state.transfers],
    })),

  updateTransfer: (id, patch) =>
    set((state) => ({
      transfers: state.transfers.map((t) =>
        t.id === id ? { ...t, ...patch } : t
      ),
    })),

  removeTransfer: (id) =>
    set((state) => ({
      transfers: state.transfers.filter((t) => t.id !== id),
    })),

  clearTransfers: () =>
    set({
      transfers: [],
    }),

  setActiveCount: (count) => set({ activeCount: count }),
  setMaxConcurrency: (max) => set({ maxConcurrency: Math.max(1, Math.min(max, 8)) }),
}));
