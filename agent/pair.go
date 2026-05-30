package main

import (
	"encoding/json"
	"net/http"

	"github.com/123100123/lanlink/internal/auth"
	"github.com/123100123/lanlink/protocol"
)

const pairingToken = "123456"

func pairHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req protocol.PairRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if req.Token != pairingToken {
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(protocol.PairResponse{
			Status: "error",
			Error:  "invalid pairing token",
		})

		return
	}

	deviceID, err := auth.GenerateToken(8)
	if err != nil {
		http.Error(w, "failed to generate device id", http.StatusInternalServerError)
		return
	}

	authToken, err := auth.GenerateToken(32)
	if err != nil {
		http.Error(w, "failed to generate auth token", http.StatusInternalServerError)
		return
	}

	response := protocol.PairResponse{
		Status:    "paired",
		DeviceID:  "device_" + deviceID,
		AuthToken: authToken,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}