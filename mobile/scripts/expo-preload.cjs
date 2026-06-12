const path = require("path");
const Module = require("module");

const originalResolveFilename = Module._resolveFilename;
const originalLoad = Module._load;

const versionsModulePath = path.join(
  __dirname,
  "..",
  "node_modules",
  "@expo",
  "cli",
  "build",
  "src",
  "api",
  "getVersions.js"
);

require.cache[versionsModulePath] = {
  id: versionsModulePath,
  filename: versionsModulePath,
  loaded: true,
  exports: {
    getVersionsAsync: async () => ({
      sdkVersions: {},
      androidVersion: "0.0.0",
      iosVersion: "0.0.0",
    }),
  },
};

Module._resolveFilename = function (request, parent, isMain, options) {
  if (
    request === "metro/src/lib/TerminalReporter" ||
    request === "metro/src/lib/TerminalReporter.js"
  ) {
    return path.join(
      __dirname,
      "..",
      "node_modules",
      "metro",
      "src",
      "lib",
      "TerminalReporter.js"
    );
  }

  return originalResolveFilename.call(this, request, parent, isMain, options);
};

Module._load = function (request, parent, isMain) {
  const metroIndexPath = path.join(
    __dirname,
    "..",
    "node_modules",
    "metro",
    "src",
    "index.js"
  );

  if (request === metroIndexPath) {
    const metro = originalLoad.call(this, request, parent, isMain);
    return {
      ...metro,
      runServer: async (...args) => {
        const result = await metro.runServer(...args);
        return result && result.httpServer ? result.httpServer : result;
      },
    };
  }

  return originalLoad.call(this, request, parent, isMain);
};
