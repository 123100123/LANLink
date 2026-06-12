# Debugging Expo Go Connection Issues

## Quick Start

```bash
cd /home/anon/dev/lanlink/mobile

# Terminal 1: Start the dev server with full logs
npx expo start

# Terminal 2: Tail the logs (on a separate terminal)
tail -f expo-logs.txt
```

## Network Requirements

**Make sure:**
1. ✅ Device and dev machine are on the **SAME WiFi network**
2. ✅ Check device IP: `ipconfig getifaddr en0` (Mac) or Settings → About (Android)
3. ✅ Check dev machine IP: `hostname -I` or look at the Expo output (exp://YOUR_IP:8081)
4. ✅ No firewall blocking port 8081

## When Connecting with Expo Go

1. Open **Expo Go** app on your device
2. Tap **Scan QR code** (or press **s** in terminal)
3. Scan the QR code shown in the Terminal
4. **Watch the Terminal for errors** - any React/TypeScript errors will appear here

## Common Errors & Fixes

### "Something went wrong"
Look for error messages in the terminal like:
- `Error: Cannot find module '@/...'` → Path alias issue
- `ReferenceError: X is not defined` → Undefined variable
- Network timeout → Device can't reach dev machine IP

### Network Connection Failed
```bash
# Check if your device can reach the dev server
# On your phone, open a browser to: http://YOUR_DEV_MACHINE_IP:8081

# Find your dev machine IP:
ifconfig | grep "inet " | grep -v 127.0.0.1
```

## View Console Logs on Device

Open the app and navigate to `/debug` route to see:
- All console.log statements
- Errors as they happen
- Warnings with full stack traces

Example:
```bash
# In the app code:
console.log("User connected:", user);  // Will appear on debug screen

// In terminal:
→ [LOG] User connected: {...}
```

## Direct Terminal Debugging

The terminal running `npx expo start` shows REAL-TIME logs:

```
Starting Metro Bundler
(node:12345) [UNDICI-EHPA] Warning: ...
▄▄▄▄▄▄▄ QR Code ▄▄▄▄▄▄▄
...
› Metro waiting on exp://192.168.1.100:8081

(When device connects:)
LOG › User opened app
LOG › Fetching data...
ERROR › Failed to fetch: Network error
```

## Save Logs to File

```bash
# Capture all output to file for analysis later
npx expo start > expo-logs.txt 2>&1 &

# View in real-time:
tail -f expo-logs.txt

# Search for errors:
grep -E "ERROR|error|Error" expo-logs.txt
```

## Next Steps

1. Try connecting device again
2. Watch the Terminal output - errors will be printed there
3. Share the **first error message** from the terminal for diagnosis
