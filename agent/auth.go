package main

import (
	"net/http"
	"strings"

	"github.com/123100123/lanlink/internal/store"
)

func authenticateRequest(r *http.Request) (*store.Device, bool) {
	header := r.Header.Get("Authorization")

	if header == "" {
		return nil, false
	}

	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return nil, false
	}

	token := strings.TrimPrefix(header, prefix)

	deviceStore, err := store.Load(deviceStorePath)
	if err != nil {
		return nil, false
	}

	device, ok := deviceStore.FindDeviceByToken(token)
	if !ok {
		return nil, false
	}

	return device, true
}