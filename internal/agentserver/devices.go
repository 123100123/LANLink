package agentserver

import (
	"encoding/json"
	"net/http"

	"github.com/123100123/lanlink/internal/paths"
	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/protocol"
)

func devicesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	deviceStore, err := store.Load(paths.DeviceStorePath)
	if err != nil {
		http.Error(
			w,
			"failed to load device store",
			http.StatusInternalServerError,
		)
		return
	}

	response := protocol.DevicesResponse{
		Devices: deviceStore.PublicDevices(),
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
	}
}
