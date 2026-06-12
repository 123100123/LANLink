import AsyncStorage from "@react-native-async-storage/async-storage";

export type Preferences = {
  agentAddress: string;
  deviceName: string;
  autoConnect: boolean;
};

const KEY = "lanlink.preferences";

const defaultPreferences: Preferences = {
  agentAddress: "",
  deviceName: "lanlink-mobile",
  autoConnect: true,
};

export async function loadPreferences(): Promise<Preferences> {
  const raw = await AsyncStorage.getItem(KEY);
  if (!raw) {
    return defaultPreferences;
  }

  return { ...defaultPreferences, ...(JSON.parse(raw) as Partial<Preferences>) };
}

export async function savePreferences(preferences: Preferences): Promise<void> {
  await AsyncStorage.setItem(KEY, JSON.stringify(preferences));
}
