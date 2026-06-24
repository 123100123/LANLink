package dashboard

import (
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strings"

	agentweb "github.com/123100123/lanlink/agent-web"
	"github.com/123100123/lanlink/internal/agentserver"
	"github.com/123100123/lanlink/internal/paths"
	"github.com/123100123/lanlink/internal/store"
)

func RegisterRoutes() {
	http.HandleFunc("/ui", indexHandler)
	http.HandleFunc("/ui/", subHandler)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if !IsLocalRequest(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	data, err := fs.ReadFile(agentweb.Files, "index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func subHandler(w http.ResponseWriter, r *http.Request) {
	if !IsLocalRequest(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/ui/")

	switch {
	case path == "state":
		s := agentserver.GetState()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)

	case path == "qr":
		QRHandler(w, r)

	case path == "settings":
		s := agentserver.CurrentSettings()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)

	case path == "settings/output-dir" && r.Method == http.MethodPost:
		handleSetOutputDir(w, r)

	case path == "settings/output-dir/reset" && r.Method == http.MethodPost:
		agentserver.ResetOutputDir()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reset", "output_dir": agentserver.GetOutputDir()})

	case path == "fs/list":
		handleFsList(w, r)

	case path == "fs/mkdir" && r.Method == http.MethodPost:
		handleFsMkdir(w, r)

	case path == "discovery/scan":
		handleDiscoveryScan(w, r)

	case path == "clients/unpair" && r.Method == http.MethodPost:
		handleUnpairClient(w, r)

	case path == "transfers/cancel" && r.Method == http.MethodPost:
		handleCancelTransfer(w, r)

	case strings.HasPrefix(path, "assets/"):
		assetPath := path
		data, err := fs.ReadFile(agentweb.Files, assetPath)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if strings.HasSuffix(assetPath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(assetPath, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(assetPath, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		w.Write(data)

	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func handleSetOutputDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "missing path"})
		return
	}

	if err := agentserver.SetOutputDir(req.Path); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "saved", "output_dir": agentserver.GetOutputDir()})
}

func handleUnpairClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DeviceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "missing device_id"})
		return
	}

	deviceStore, err := store.Load(paths.DeviceStorePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "failed to load device store"})
		return
	}

	found := deviceStore.RemoveDevice(req.DeviceID)
	if !found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "device not found"})
		return
	}

	if err := deviceStore.Save(paths.DeviceStorePath); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "failed to save device store"})
		return
	}

	agentserver.RemovePairedClient(req.DeviceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleCancelTransfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TransferID string `json:"transfer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TransferID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "missing transfer_id"})
		return
	}

	agentserver.CancelTransfer(req.TransferID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "transfer_id": req.TransferID})
}

func IsLocalRequest(r *http.Request) bool {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}

	if remoteHost == "127.0.0.1" || remoteHost == "::1" || remoteHost == "localhost" {
		return true
	}

	remoteIP := net.ParseIP(remoteHost)
	if remoteIP != nil && remoteIP.IsLoopback() {
		return true
	}

	return false
}
