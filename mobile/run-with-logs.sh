#!/bin/bash
echo "Starting Expo with full logging to expo-logs.txt..."
npx expo start 2>&1 | tee expo-logs.txt
