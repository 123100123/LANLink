package ws

import (
	"fmt"
	"log"
	"time"

	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func authenticate(conn *websocket.Conn, authToken string) {
	authPayload, err := protocol.EncodePayload(protocol.AuthRequest{
		Token: authToken,
	})
	if err != nil {
		log.Fatal("failed to encode auth payload:", err)
	}

	authMessage := protocol.Message{
		Type:      "auth",
		ID:        "auth_1",
		Timestamp: time.Now().UnixMilli(),
		Payload:   authPayload,
	}

	err = conn.WriteJSON(authMessage)
	if err != nil {
		log.Fatal("failed to send auth message:", err)
	}

	var authResponse protocol.Message

	err = conn.ReadJSON(&authResponse)
	if err != nil {
		log.Fatal("failed to read auth response:", err)
	}

	if authResponse.Type == "auth.failed" {
		var failed protocol.AuthFailed

		err = protocol.DecodePayload(authResponse.Payload, &failed)
		if err != nil {
			log.Fatal("failed to decode auth failure:", err)
		}

		log.Fatal("websocket authentication failed:", failed.Error)
	}

	if authResponse.Type != "auth.success" {
		log.Fatal("unexpected websocket response:", authResponse.Type)
	}

	var success protocol.AuthSuccess

	err = protocol.DecodePayload(authResponse.Payload, &success)
	if err != nil {
		log.Fatal("failed to decode auth success:", err)
	}

	fmt.Println("WebSocket connected")
	fmt.Println("Authenticated")
	fmt.Println("Device ID:", success.DeviceID)
	fmt.Println("Device Name:", success.DeviceName)
}