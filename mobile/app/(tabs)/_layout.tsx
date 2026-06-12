import { Ionicons } from "@expo/vector-icons";
import { Tabs } from "expo-router";

import { useSessionStore } from "@/store/sessionStore";

export default function TabsLayout() {
  const credentials = useSessionStore((state) => state.credentials);

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: {
          backgroundColor: "#0b1220",
          borderTopColor: "#1d2a44",
          height: 64,
          paddingBottom: 8,
          paddingTop: 8,
        },
        tabBarActiveTintColor: "#ffffff",
        tabBarInactiveTintColor: "#7d8aa5",
        tabBarLabelStyle: {
          fontSize: 12,
          fontWeight: "600",
        },
      }}
    >
      <Tabs.Screen
        name="home"
        options={{
          href: null,
        }}
      />

      <Tabs.Screen
        name="devices/index"
        options={{
          href: null,
        }}
      />

      <Tabs.Screen
        name="devices/[deviceId]"
        options={{
          title: "Device",
          href: credentials?.deviceId
            ? `/(tabs)/devices/${credentials.deviceId}`
            : null,
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons
              name={focused ? "phone-portrait" : "phone-portrait-outline"}
              color={color}
              size={size}
            />
          ),
        }}
      />

      <Tabs.Screen
        name="transfers"
        options={{
          title: "Transfers",
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons
              name={focused ? "swap-horizontal" : "swap-horizontal-outline"}
              color={color}
              size={size}
            />
          ),
        }}
      />

      <Tabs.Screen
        name="settings"
        options={{
          title: "Settings",
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons
              name={focused ? "settings" : "settings-outline"}
              color={color}
              size={size}
            />
          ),
        }}
      />
    </Tabs>
  );
}