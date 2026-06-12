import { create } from "zustand";

type TransferState = {
  currentTransferId: string | null;
  progress: number;
  status: string;
  error: string | null;
  setTransfer: (transferId: string | null) => void;
  setProgress: (progress: number) => void;
  setStatus: (status: string) => void;
  setError: (error: string | null) => void;
};

export const useTransferStore = create<TransferState>((set) => ({
  currentTransferId: null,
  progress: 0,
  status: "idle",
  error: null,
  setTransfer: (currentTransferId) => set({ currentTransferId }),
  setProgress: (progress) => set({ progress }),
  setStatus: (status) => set({ status }),
  setError: (error) => set({ error }),
}));
