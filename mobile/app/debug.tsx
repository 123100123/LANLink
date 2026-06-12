import { View, Text, ScrollView, StyleSheet } from "react-native";
import { useEffect, useState } from "react";

export default function DebugScreen() {
  const [logs, setLogs] = useState<string[]>([]);

  useEffect(() => {
    // Capture console.log
    const originalLog = console.log;
    const originalError = console.error;
    const originalWarn = console.warn;

    const addLog = (type: string, args: any[]) => {
      const message = args.map(arg => 
        typeof arg === 'string' ? arg : JSON.stringify(arg)
      ).join(' ');
      setLogs(prev => [...prev, `[${type}] ${message}`].slice(-50));
    };

    console.log = (...args) => {
      originalLog(...args);
      addLog('LOG', args);
    };

    console.error = (...args) => {
      originalError(...args);
      addLog('ERROR', args);
    };

    console.warn = (...args) => {
      originalWarn(...args);
      addLog('WARN', args);
    };

    return () => {
      console.log = originalLog;
      console.error = originalError;
      console.warn = originalWarn;
    };
  }, []);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Debug Logs</Text>
      <ScrollView style={styles.logContainer}>
        {logs.map((log, i) => (
          <Text key={i} style={styles.logText}>
            {log}
          </Text>
        ))}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#1e1e1e',
    padding: 10,
  },
  title: {
    color: '#fff',
    fontSize: 18,
    fontWeight: 'bold',
    marginBottom: 10,
  },
  logContainer: {
    flex: 1,
    backgroundColor: '#0d0d0d',
    borderRadius: 4,
    padding: 8,
  },
  logText: {
    color: '#0f0',
    fontSize: 11,
    fontFamily: 'monospace',
    marginBottom: 2,
  },
});
