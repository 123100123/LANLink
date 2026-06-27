const { withAndroidManifest, AndroidConfig } = require("@expo/config-plugins");

// Registers LANLink as a target in the Android system share sheet by adding
// ACTION_SEND / ACTION_SEND_MULTIPLE intent filters (any MIME type) to
// MainActivity. Runs at prebuild; pairs with the lanlink-share native module
// which reads the shared URIs at runtime.
const SHARE_ACTIONS = [
  "android.intent.action.SEND",
  "android.intent.action.SEND_MULTIPLE",
];

function buildFilter(action) {
  return {
    action: [{ $: { "android:name": action } }],
    category: [{ $: { "android:name": "android.intent.category.DEFAULT" } }],
    data: [{ $: { "android:mimeType": "*/*" } }],
  };
}

function hasFilterFor(filters, action) {
  return filters.some((filter) =>
    (filter.action || []).some((a) => a.$ && a.$["android:name"] === action)
  );
}

const withShareIntent = (config) =>
  withAndroidManifest(config, (cfg) => {
    const mainActivity = AndroidConfig.Manifest.getMainActivityOrThrow(
      cfg.modResults
    );
    mainActivity["intent-filter"] = mainActivity["intent-filter"] || [];

    for (const action of SHARE_ACTIONS) {
      if (!hasFilterFor(mainActivity["intent-filter"], action)) {
        mainActivity["intent-filter"].push(buildFilter(action));
      }
    }
    return cfg;
  });

module.exports = withShareIntent;
