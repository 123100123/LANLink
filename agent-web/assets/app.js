let lastToken = "";
let lastAddress = "";

let browserPath = "";
let browserParent = "";
let browserSelected = "";

function escapeHtml(value) {
  return String(value == null ? "" : value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

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
  if (seconds < 3600)
    return Math.floor(seconds / 60) + "m " + Math.round(seconds % 60) + "s";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return h + "h " + m + "m";
}

function formatTime(ts) {
  if (!ts) return "";
  return new Date(ts * 1000).toLocaleTimeString();
}

function formatETA(total, received, speed) {
  if (!total || !speed || received >= total) return "";
  const remaining = total - received;
  const seconds = remaining / speed;
  if (seconds < 1) return "<1s";
  if (seconds < 60) return Math.round(seconds) + "s";
  const m = Math.floor(seconds / 60);
  const s = Math.round(seconds % 60);
  return m + "m " + s + "s";
}

function copyText(elementId) {
  var el = document.getElementById(elementId);
  if (!el) return;
  navigator.clipboard.writeText(el.textContent).then(function () {
    showToast("Copied!");
  });
}

function showToast(msg) {
  var toast = document.getElementById("toast");
  if (!toast) {
    toast = document.createElement("div");
    toast.id = "toast";
    toast.className = "copied-toast";
    document.body.appendChild(toast);
  }
  toast.textContent = msg;
  toast.classList.add("show");
  clearTimeout(showToast._t);
  showToast._t = setTimeout(function () {
    toast.classList.remove("show");
  }, 1800);
}

function showSettingsStatus(msg, isError) {
  var el = document.getElementById("settings-status");
  if (!el) return;
  el.textContent = msg;
  el.style.color = isError ? "#ff7b7b" : "#5ed39b";
  setTimeout(function () {
    el.textContent = "";
  }, 3000);
}

async function saveOutputDir() {
  var input = document.getElementById("output-dir-input");
  var path = input ? input.value.trim() : "";
  if (!path) {
    showSettingsStatus("Enter a folder path", true);
    return;
  }
  try {
    var resp = await fetch("/ui/settings/output-dir", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: path }),
    });
    var data = await resp.json();
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
    var resp = await fetch("/ui/settings/output-dir/reset", { method: "POST" });
    var data = await resp.json();
    if (data.status === "reset") {
      showSettingsStatus("Reset to default", false);
      refresh();
    }
  } catch (e) {
    showSettingsStatus("Request failed", true);
  }
}

async function unpairClient(deviceId) {
  if (!confirm("Unpair this client? The device will need to pair again.")) return;
  try {
    var resp = await fetch("/ui/clients/unpair", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ device_id: deviceId }),
    });
    var data = await resp.json();
    if (data.status === "ok") {
      showToast("Client unpaired");
      refresh();
    } else {
      showToast(data.error || "Failed to unpair");
    }
  } catch (e) {
    showToast("Request failed");
  }
}

async function cancelTransfer(transferId) {
  if (!confirm("Cancel this transfer?")) return;
  try {
    var resp = await fetch("/ui/transfers/cancel", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ transfer_id: transferId }),
    });
    var data = await resp.json();
    if (data.status === "ok") {
      showToast("Transfer cancelled");
      refresh();
    } else {
      showToast(data.error || "Failed to cancel");
    }
  } catch (e) {
    showToast("Request failed");
  }
}

/* ---------- Folder browser ---------- */

function openBrowser() {
  document.getElementById("browser-overlay").classList.add("show");
  var start = document.getElementById("output-dir-input").value.trim();
  browserLoad(start);
}

function closeBrowser() {
  document.getElementById("browser-overlay").classList.remove("show");
}

function onBrowserOverlayClick(e) {
  if (e.target === document.getElementById("browser-overlay")) closeBrowser();
}

async function browserLoad(path) {
  var list = document.getElementById("browser-list");
  var url = "/ui/fs/list";
  if (path) url += "?path=" + encodeURIComponent(path);
  try {
    var resp = await fetch(url, { cache: "no-store" });
    var data = await resp.json();
    if (!resp.ok || data.status === "error") {
      list.innerHTML =
        '<p class="empty">' + escapeHtml(data.error || "Cannot open folder") + "</p>";
      return;
    }
    renderBrowser(data);
  } catch (e) {
    list.innerHTML = '<p class="empty">Request failed</p>';
  }
}

function renderBrowser(data) {
  browserPath = data.path || "";
  browserParent = data.parent || "";
  browserSelected = data.path || "";

  var pathEl = document.getElementById("browser-path");
  pathEl.textContent = data.path || "—";
  pathEl.title = data.path || "";
  document.getElementById("browser-selected").textContent = data.path
    ? "Selected: " + data.path
    : "";

  document.getElementById("browser-up").disabled = !browserParent;

  var quick = document.getElementById("browser-quick");
  quick.innerHTML = (data.quick || [])
    .map(function (q) {
      return (
        '<button class="quick-chip" data-path="' +
        escapeHtml(q.path) +
        '">' +
        escapeHtml(q.name) +
        "</button>"
      );
    })
    .join("");

  var list = document.getElementById("browser-list");
  if (!data.entries || data.entries.length === 0) {
    list.innerHTML = '<p class="empty">No subfolders here</p>';
    return;
  }
  list.innerHTML = data.entries
    .map(function (e) {
      return (
        '<div class="browser-row" data-path="' +
        escapeHtml(e.path) +
        '"><span class="folder-icon">📁</span><span>' +
        escapeHtml(e.name) +
        "</span></div>"
      );
    })
    .join("");
}

function browserUp() {
  if (browserParent) browserLoad(browserParent);
}

async function createFolder() {
  var input = document.getElementById("browser-newfolder-input");
  var name = input.value.trim();
  if (!name) return;
  try {
    var resp = await fetch("/ui/fs/mkdir", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: browserPath, name: name }),
    });
    var data = await resp.json();
    if (resp.ok && data.status === "ok") {
      input.value = "";
      browserLoad(data.path);
    } else {
      showToast(data.error || "Could not create folder");
    }
  } catch (e) {
    showToast("Request failed");
  }
}

function useThisFolder() {
  var chosen = browserSelected || browserPath;
  if (!chosen) return;
  document.getElementById("output-dir-input").value = chosen;
  closeBrowser();
  showToast("Folder selected — click Save to apply");
}

/* ---------- Render dashboard ---------- */

function renderPairedClients(clients) {
  var el = document.getElementById("paired-clients-list");
  if (!el) return;
  if (!clients || clients.length === 0) {
    el.innerHTML = '<p class="empty">No clients paired this run</p>';
    return;
  }

  var html = "";
  for (var i = 0; i < clients.length; i++) {
    var c = clients[i];
    html +=
      '<div class="client-row">' +
      '<span class="client-name">' + escapeHtml(c.device_name) + "</span>" +
      '<span class="client-id">' + escapeHtml(c.device_id) + "</span>" +
      '<span class="client-time">' + formatTime(c.paired_at) + "</span>" +
      '<button class="icon-btn danger" title="Unpair client" onclick="unpairClient(\'' +
      escapeHtml(c.device_id).replace(/'/g, "\\'") +
      "')\">&times;</button>" +
      "</div>";
  }
  el.innerHTML = html;
}

function renderTransferItem(t) {
  var statusClass =
    t.status === "receiving"
      ? "status-receiving"
      : t.status === "saved"
        ? "status-saved"
        : t.status === "cancelled"
          ? "status-cancelled"
          : "status-failed";

  var name = escapeHtml(t.filename);
  var status = escapeHtml(t.status);

  if (t.status === "receiving") {
    var pct = t.total > 0 ? ((t.received / t.total) * 100).toFixed(1) : "";
    var percentText = pct ? pct + "%" : "";
    var barHtml = "";
    if (t.total > 0) {
      var pctNum = parseFloat(pct) || 0;
      barHtml =
        '<div class="progress-track"><div class="progress-fill" style="width:' +
        pctNum +
        '%"></div></div>';
    }

    var details = formatBytes(t.received);
    if (t.total > 0) details += " / " + formatBytes(t.total);
    if (t.speed > 0) details += " · " + formatSpeed(t.speed);
    var eta = formatETA(t.total, t.received, t.speed);
    if (eta) details += " · ETA " + eta;

    var cancelBtn = "";
    if (t.cancellable) {
      cancelBtn =
        '<button class="icon-btn danger" title="Cancel transfer" onclick="cancelTransfer(\'' +
        escapeHtml(t.id).replace(/'/g, "\\'") +
        "')\">&times;</button>";
    }

    return (
      '<div class="transfer-item">' +
        '<div class="transfer-top">' +
          '<span class="transfer-filename">' + name + "</span>" +
          cancelBtn +
          '<span class="transfer-status ' + statusClass + '">' + status + "</span>" +
        "</div>" +
        '<div class="transfer-progress-row">' +
          barHtml +
          (percentText
            ? '<span class="transfer-percent">' + percentText + "</span>"
            : "") +
        "</div>" +
        '<div class="transfer-details">' + escapeHtml(details) + "</div>" +
      "</div>"
    );
  }

  var sizeInfo = formatBytes(t.received);
  if (t.status === "saved" && t.total > 0) sizeInfo = formatBytes(t.total);

  var details = sizeInfo;
  if (t.speed > 0) details += " · " + formatSpeed(t.speed);
  if (t.status === "saved" && t.completed_at) {
    details += " · " + formatTime(t.completed_at);
  }

  var extra = "";
  if (t.status === "saved" && t.path) {
    extra = '<div class="transfer-path">' + escapeHtml(t.path) + "</div>";
  }
  if (t.status === "failed" && t.error) {
    extra = '<div class="transfer-error">' + escapeHtml(t.error) + "</div>";
  }
  if (t.status === "cancelled") {
    extra = '<div class="transfer-error">Cancelled by user</div>';
  }

  return (
    '<div class="transfer-item">' +
      '<div class="transfer-top">' +
        '<span class="transfer-filename">' + name + "</span>" +
        '<span class="transfer-status ' + statusClass + '">' + status + "</span>" +
      "</div>" +
      '<div class="transfer-details">' + escapeHtml(details) + "</div>" +
      extra +
    "</div>"
  );
}

function render(state) {
  var badge = document.getElementById("status-badge");
  var badgeText = document.getElementById("status-badge-text");
  if (state.status === "ok") {
    badgeText.textContent = "Running";
    badge.className = "badge ok";
  } else {
    badgeText.textContent = state.status || "Unknown";
    badge.className = "badge error";
  }

  document.getElementById("address").textContent = state.address || "—";
  document.getElementById("token").textContent = state.token || "—";
  document.getElementById("agent-status").textContent = state.status || "—";
  document.getElementById("uptime").textContent = formatUptime(
    state.uptime_seconds || 0
  );
  document.getElementById("files-received").textContent = String(
    state.received_count || 0
  );
  document.getElementById("active-transfers").textContent = String(
    state.active_count || 0
  );
  document.getElementById("output-dir").textContent =
    state.output_dir || "received";

  var qr = document.getElementById("pairing-qr");
  if (state.token !== lastToken || state.address !== lastAddress) {
    qr.src =
      "/ui/qr?t=" +
      encodeURIComponent(state.token) +
      "&a=" +
      encodeURIComponent(state.address);
    lastToken = state.token;
    lastAddress = state.address;
  }

  renderPairedClients(state.paired_clients);

  var activeList = document.getElementById("active-list");
  var active = (state.transfers || []).filter(function (t) {
    return t.status === "receiving";
  });
  if (active.length === 0) {
    activeList.innerHTML = '<p class="empty">No active transfers</p>';
  } else {
    activeList.innerHTML = active.map(renderTransferItem).join("");
  }

  var receivedList = document.getElementById("received-list");
  var received = (state.transfers || []).filter(function (t) {
    return t.status === "saved" || t.status === "cancelled" || t.status === "failed";
  });
  if (received.length === 0) {
    receivedList.innerHTML = '<p class="empty">No files received yet</p>';
  } else {
    receivedList.innerHTML = received.map(renderTransferItem).join("");
  }
}

async function refresh() {
  try {
    var response = await fetch("/ui/state", { cache: "no-store" });
    if (response.ok) {
      var state = await response.json();
      render(state);
    }
  } catch (e) {
    // silently retry
  }
}

/* ---------- Init ---------- */

document.getElementById("browser-quick").addEventListener("click", function (e) {
  var chip = e.target.closest(".quick-chip");
  if (chip && chip.dataset.path) browserLoad(chip.dataset.path);
});
document.getElementById("browser-list").addEventListener("click", function (e) {
  var row = e.target.closest(".browser-row");
  if (row && row.dataset.path) browserLoad(row.dataset.path);
});
document.getElementById("browser-newfolder-input").addEventListener("keydown", function (e) {
  if (e.key === "Enter") createFolder();
});
document.addEventListener("keydown", function (e) {
  if (e.key === "Escape") closeBrowser();
});

setInterval(refresh, 1000);
refresh();
