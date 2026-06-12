import { create } from "zustand";

import type { DeviceInfo } from "@/lib/protocol/payloads";

type DevicesState = {
  devices: DeviceInfo[];
  setDevices: (devices: DeviceInfo[]) => void;
};

export const useDevicesStore = create<DevicesState>((set) => ({
  devices: [],
  setDevices: (devices) => set({ devices }),
}));
