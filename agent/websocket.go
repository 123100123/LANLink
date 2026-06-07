package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	log.Println("websocket client connected")

	device, ok := authenticateWebSocket(conn)
	if !ok {
		log.Println("websocket authentication failed")
		return
	}

	log.Println("websocket authenticated:", device.DeviceID)

	// Phase 4A stops here.
	// Future phases will add a read loop here for ping, commands, media, etc.
}