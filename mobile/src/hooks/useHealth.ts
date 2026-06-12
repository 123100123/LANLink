import { useState } from "react";

type HealthState = {
  loading: boolean;
  message: string;
};

export function useHealth() {
  return useState<HealthState>({
    loading: false,
    message: "Idle",
  });
}
