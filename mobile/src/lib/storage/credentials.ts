import { Platform } from "react-native";
import AsyncStorage from "@react-native-async-storage/async-storage";

// Only import SecureStore if not on web
let SecureStore: typeof import("expo-secure-store") | null = null;
if (Platform.OS !== "web") {
  SecureStore = require("expo-secure-store");
}

export type Credentials = {
  agentAddress: string;
  deviceId: string;
  authToken: string;
};

const KEY = "lanlink.credentials";

// Use SecureStore on native, AsyncStorage on web
async function secureSetItem(key: string, value: string): Promise<void> {
  if (SecureStore && Platform.OS !== "web") {
    await SecureStore.setItemAsync(key, value);
  } else {
    await AsyncStorage.setItem(key, value);
  }
}

async function secureGetItem(key: string): Promise<string | null> {
  if (SecureStore && Platform.OS !== "web") {
    return await SecureStore.getItemAsync(key);
  } else {
    return await AsyncStorage.getItem(key);
  }
}

async function secureDeleteItem(key: string): Promise<void> {
  if (SecureStore && Platform.OS !== "web") {
    await SecureStore.deleteItemAsync(key);
  } else {
    await AsyncStorage.removeItem(key);
  }
}

export async function saveCredentials(credentials: Credentials): Promise<void> {
  await secureSetItem(KEY, JSON.stringify(credentials));
}

export async function loadCredentials(): Promise<Credentials | null> {
  try {
    const value = await secureGetItem(KEY);
    if (!value) {
      return null;
    }
    return JSON.parse(value) as Credentials;
  } catch (error) {
    console.error("Failed to load credentials:", error);
    return null;
  }
}

export async function clearCredentials(): Promise<void> {
  await secureDeleteItem(KEY);
}
