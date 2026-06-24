package agentserver

import (
	"encoding/json"
	"net/http"

	"github.com/123100123/lanlink/internal/auth"
	"github.com/123100123/lanlink/internal/paths"
	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/protocol"
)

// pairAutoHandler issues credentials WITHOUT a pairing token. It is the open,
// LAN-only auto-connect endpoint used by discovery: any client that can reach
// the receiver over the local network obtains credentials. This is a deliberate
// trust relaxation for trusted LANs; the token + QR pairing flow at /pair is
// unaffected, and auto-connected clients appear in the dashboard with unpair.
func pairAutoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req protocol.PairRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body optional; any token is ignored

	w.Header().Set("Content-Type", "application/json")

	deviceIDRaw, err := auth.GenerateToken(8)
	if err != nil {
		writePairAutoError(w, "failed to generate device id")
		return
	}
	authToken, err := auth.GenerateToken(32)
	if err != nil {
		writePairAutoError(w, "failed to generate auth token")
		return
	}

	deviceName := req.DeviceName
	if deviceName == "" {
		deviceName = "auto-device"
	}

	device := store.Device{
		DeviceID:   "device_" + deviceIDRaw,
		DeviceName: deviceName,
		AuthToken:  authToken,
	}

	deviceStore, err := store.Load(paths.DeviceStorePath)
	if err != nil {
		writePairAutoError(w, "failed to load device store")
		return
	}
	deviceStore.AddDevice(device)
	if err := deviceStore.Save(paths.DeviceStorePath); err != nil {
		writePairAutoError(w, "failed to save device")
		return
	}

	json.NewEncoder(w).Encode(protocol.PairResponse{
		Status:    "paired",
		DeviceID:  device.DeviceID,
		AuthToken: device.AuthToken,
	})

	AddPairedClient(device.DeviceID, device.DeviceName)
}

func writePairAutoError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(protocol.PairResponse{Status: "error", Error: msg})
}
