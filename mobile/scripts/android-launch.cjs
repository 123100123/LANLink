const { spawnSync, spawn } = require("child_process");
const path = require("path");

function hasAdb() {
  const result = spawnSync("adb", ["version"], {
    stdio: "ignore",
  });
  return result.status === 0;
}

function run() {
  const args = [
    "-r",
    path.join(__dirname, "expo-preload.cjs"),
    path.join(__dirname, "..", "node_modules", "@expo", "cli", "build", "bin", "cli"),
    "start",
    "--offline",
  ];

  if (hasAdb()) {
    args.push("--android");
  } else {
    console.log("Android SDK not found. Starting Metro without the Android handoff.");
  }

  const child = spawn(process.execPath, args, {
    stdio: "inherit",
    env: {
      ...process.env,
      EXPO_NO_TYPESCRIPT_SETUP: "1",
    },
  });

  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exit(code ?? 0);
  });
}

run();
