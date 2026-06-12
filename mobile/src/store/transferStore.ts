import { create } from "zustand";

export type TransferStatus = "uploading" | "completed" | "failed";

export type TransferItem = {
  id: string;
  filename: string;
  size?: number;
  sentBytes: number;
  progress: number;
  status: TransferStatus;
  startedAt: number;
  completedAt?: number;
  error?: string;
  savedPath?: string;
};

type TransferStore = {
  transfers: TransferItem[];
  addTransfer: (transfer: TransferItem) => void;
  updateTransfer: (
    id: string,
    patch: Partial<Omit<TransferItem, "id">>
  ) => void;
  clearTransfers: () => void;
};

export const useTransferStore = create<TransferStore>((set) => ({
  transfers: [],

  addTransfer: (transfer) =>
    set((state) => ({
      transfers: [transfer, ...state.transfers],
    })),

  updateTransfer: (id, patch) =>
    set((state) => ({
      transfers: state.transfers.map((transfer) =>
        transfer.id === id ? { ...transfer, ...patch } : transfer
      ),
    })),

  clearTransfers: () =>
    set({
      transfers: [],
    }),
}));