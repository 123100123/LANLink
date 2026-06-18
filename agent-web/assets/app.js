let lastToken = "";
let lastAddress = "";

function formatBytes(bytes) {
  if (bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let i = 0;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i++;
  }
  return value.toFixed(i === 0 ? 0 : 1) + " " + units[i];
}

function formatSpeed(bytesPerSec) {
  if (bytesPerSec <= 0) return "";
  const mbps = bytesPerSec / 1024 / 1024;
  return mbps.toFixed(1) + " MB/s";
}

function formatUptime(seconds) {
  if (seconds < 60) return Math.round(seconds) + "s";
  if (seconds < 3600) return Math.floor(seconds / 60) + "m " + Math.round(seconds % 60) + "s";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return h + "h " + m + "m";
}

function formatTime(ts) {
  if (!ts) return "";
  return new Date(ts * 1000).toLocaleTimeString();
}

function copyText(elementId) {
  const el = document.getElementById(elementId);
  if (!el) return;
  navigator.clipboard.writeText(el.textContent).then(() => {
    showToast("Copied!");
  });
}

function showToast(msg) {
  let toast = document.getElementById("toast");
  if (!toast) {
    toast = document.createElement("div");
    toast.id = "toast";
    toast.className = "copied-toast";
    document.body.appendChild(toast);
  }
  toast.textContent = msg;
  toast.classList.add("show");
  setTimeout(() => toast.classList.remove("show"), 1500);
}

function showSettingsStatus(msg, isError) {
  const el = document.getElementById("settings-status");
  if (!el) return;
  el.textContent = msg;
  el.style.color = isError ? "#ff8a8a" : "#6fcf97";
  setTimeout(() => { el.textContent = ""; }, 3000);
}

async function saveOutputDir() {
  const input = document.getElementById("output-dir-input");
  const path = input ? input.value.trim() : "";
  if (!path) {
    showSettingsStatus("Enter a folder path", true);
    return;
  }

  try {
    const resp = await fetch("/ui/settings/output-dir", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path }),
    });
    const data = await resp.json();
    if (data.status === "saved") {
      showSettingsStatus("Output folder saved", false);
      input.value = "";
      refresh();
    } else {
      showSettingsStatus(data.error || "Failed to save", true);
    }
  } catch (e) {
    showSettingsStatus("Request failed", true);
  }
}

async function resetOutputDir() {
  try {
    const resp = await fetch("/ui/settings/output-dir/reset", {
      method: "POST",
    });
    const data = await resp.json();
    if (data.status === "reset") {
      showSettingsStatus("Reset to default", false);
      refresh();
    }
  } catch (e) {
    showSettingsStatus("Request failed", true);
  }
}

function renderTransferItem(t) {
  const percent = t.total > 0 ? Math.round((t.received / t.total) * 100) : 0;
  const statusClass = t.status === "receiving" ? "status-receiving"
    : t.status === "saved" ? "status-saved"
    : "status-failed";

  let meta = formatBytes(t.received);
  if (t.total > 0) meta += " / " + formatBytes(t.total);
  if (t.speed > 0) meta += " \u00b7 " + formatSpeed(t.speed);

  let extra = "";
  if (t.status === "saved" && t.path) {
    extra = '<div class="transfer-meta" style="margin-top:4px;color:#6f7f9d;font-size:11px">' + t.path + "</div>";
  }
  if (t.status === "failed" && t.error) {
    extra = '<div class="transfer-meta" style="margin-top:4px;color:#ff8a8a;font-size:11px">' + t.error + "</div>";
  }

  let progressHtml = "";
  if (t.status === "receiving" && t.total > 0) {
    progressHtml = '<div class="progress-track"><div class="progress-fill" style="width:' + percent + '%"></div></div>';
  }

  return (
    '<div class="transfer-item">' +
      '<span class="transfer-filename">' + t.filename + "</span>" +
      progressHtml +
      '<span class="transfer-meta">' + meta + "</span>" +
      '<span class="transfer-status ' + statusClass + '">' + t.status + "</span>" +
    "</div>" +
    extra
  );
}

function render(state) {
  const badge = document.getElementById("status-badge");
  if (state.status === "ok") {
    badge.textContent = "Running";
    badge.className = "badge ok";
  } else {
    badge.textContent = state.status || "Unknown";
    badge.className = "badge error";
  }

  document.getElementById("address").textContent = state.address || "\u2014";
  document.getElementById("token").textContent = state.token || "\u2014";
  document.getElementById("agent-status").textContent = state.status || "\u2014";
  document.getElementById("uptime").textContent = formatUptime(state.uptime_seconds || 0);
  document.getElementById("files-received").textContent = (state.received_count || 0).toString();
  document.getElementById("active-transfers").textContent = (state.active_count || 0).toString();
  document.getElementById("output-dir").textContent = state.output_dir || "received";

  const qr = document.getElementById("pairing-qr");
  if (state.token !== lastToken || state.address !== lastAddress) {
    qr.src = "/ui/qr?t=" + encodeURIComponent(state.token) + "&a=" + encodeURIComponent(state.address);
    lastToken = state.token;
    lastAddress = state.address;
  }

  const activeList = document.getElementById("active-list");
  const active = (state.transfers || []).filter(function(t) {
    return t.status === "receiving";
  });
  if (active.length === 0) {
    activeList.innerHTML = '<p class="empty">No active transfers</p>';
  } else {
    activeList.innerHTML = active.map(renderTransferItem).join("");
  }

  const receivedList = document.getElementById("received-list");
  const received = (state.transfers || []).filter(function(t) {
    return t.status === "saved";
  });
  if (received.length === 0) {
    receivedList.innerHTML = '<p class="empty">No files received yet</p>';
  } else {
    receivedList.innerHTML = received.map(renderTransferItem).join("");
  }
}

async function refresh() {
  try {
    const response = await fetch("/ui/state", { cache: "no-store" });
    if (response.ok) {
      const state = await response.json();
      render(state);
    }
  } catch (e) {
    // silently retry on next interval
  }
}

setInterval(refresh, 1000);
refresh();
