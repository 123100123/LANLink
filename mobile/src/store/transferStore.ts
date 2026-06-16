import { create } from "zustand";

export type TransferStatus =
  | "waiting"
  | "uploading"
  | "completed"
  | "failed"
  | "cancelled";

export type TransferItem = {
  id: string;
  uri: string;
  tempUri?: string;
  filename: string;
  size: number;
  agentAddress: string;
  authToken: string;
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
  addTransfer: (transfer: TransferItem) => void;
  addTransfers: (transfers: TransferItem[]) => void;
  updateTransfer: (
    id: string,
    patch: Partial<Omit<TransferItem, "id">>
  ) => void;
  removeTransfer: (id: string) => void;
  clearCompleted: () => void;
  clearAll: () => void;
};

export const useTransferStore = create<TransferStore>((set) => ({
  transfers: [],

  addTransfer: (transfer) =>
    set((state) => ({
      transfers: [...state.transfers, transfer],
    })),

  addTransfers: (newTransfers) =>
    set((state) => ({
      transfers: [...state.transfers, ...newTransfers],
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

  clearCompleted: () =>
    set((state) => ({
      transfers: state.transfers.filter((t) => t.status !== "completed"),
    })),

  clearAll: () => set({ transfers: [] }),
}));
