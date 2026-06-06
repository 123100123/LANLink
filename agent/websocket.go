package main

import (
	"log"
	"net/http"
	"time"

	"github.com/123100123/lanlink/protocol"
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

	var msg protocol.Message

	err = conn.ReadJSON(&msg)
	if err != nil {
		log.Println("failed to read websocket message:", err)
		return
	}

	log.Println("received websocket message:", msg.Type)

	response := protocol.Message{
		Type:      "hello.response",
		ID:        msg.ID,
		Timestamp: time.Now().Unix(),
		Payload:   "hello from lanlink agent",
	}

	err = conn.WriteJSON(response)
	if err != nil {
		log.Println("failed to write websocket response:", err)
		return
	}

	log.Println("websocket response sent")
}