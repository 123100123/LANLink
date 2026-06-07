package main

import (
	"log"
	"time"

	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func handleWebSocketSession(conn *websocket.Conn, device *store.Device) {
	for {
		var msg protocol.Message

		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("websocket read stopped for", device.DeviceID, ":", err)
			return
		}

		log.Println("websocket message from", device.DeviceID, ":", msg.Type)

		switch msg.Type {
		case "hello":
			handleWebSocketHello(conn, msg)

		default:
			writeWebSocketError(conn, msg.ID, "unknown message type")
		}
	}
}

func handleWebSocketHello(conn *websocket.Conn, msg protocol.Message) {
	payload, err := protocol.EncodePayload("hello from authenticated lanlink agent")
	if err != nil {
		return
	}

	response := protocol.Message{
		Type:      "hello.response",
		ID:        msg.ID,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	conn.WriteJSON(response)
}

func writeWebSocketError(conn *websocket.Conn, id string, reason string) {
	payload, err := protocol.EncodePayload(reason)
	if err != nil {
		return
	}

	response := protocol.Message{
		Type:      "error",
		ID:        id,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	conn.WriteJSON(response)
}