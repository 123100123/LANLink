package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/123100123/lanlink/agent/dashboard"
	"github.com/123100123/lanlink/internal/auth"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/paths"
	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/protocol"
)

func pairHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var req protocol.PairRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if pairingManager == nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(protocol.PairResponse{
			Status: "error",
			Error:  "pairing manager is not initialized",
		})

		return
	}

	if !pairingManager.Validate(req.Token) {
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(protocol.PairResponse{
			Status: "error",
			Error:  "invalid pairing token",
		})

		return
	}

	deviceIDRaw, err := auth.GenerateToken(8)
	if err != nil {
		http.Error(
			w,
			"failed to generate device id",
			http.StatusInternalServerError,
		)
		return
	}

	authToken, err := auth.GenerateToken(32)
	if err != nil {
		http.Error(
			w,
			"failed to generate auth token",
			http.StatusInternalServerError,
		)
		return
	}

	deviceName := req.DeviceName
	if deviceName == "" {
		deviceName = "unknown-device"
	}

	device := store.Device{
		DeviceID:   "device_" + deviceIDRaw,
		DeviceName: deviceName,
		AuthToken:  authToken,
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

	deviceStore.AddDevice(device)

	err = deviceStore.Save(paths.DeviceStorePath)
	if err != nil {
		http.Error(
			w,
			"failed to save device",
			http.StatusInternalServerError,
		)
		return
	}

	response := protocol.PairResponse{
		Status:    "paired",
		DeviceID:  device.DeviceID,
		AuthToken: device.AuthToken,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
		return
	}

	dashboard.AddPairedClient(device.DeviceID, device.DeviceName)

	if err := pairingManager.Rotate(); err != nil {
		log.Println("failed to rotate pairing token:", err)
		return
	}

	newToken := pairingManager.Token()
	dashboard.SetToken(newToken)
	log.Println("New pairing token:", newToken)
	log.Println("Use this token to pair a new device.")
	log.Println("A new token will be generated after each successful pairing.")
	log.Println("")

	cfg := config.Load()
	printPairingQR(newToken, cfg.Port)
}
