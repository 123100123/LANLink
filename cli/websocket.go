package main

import (
	"fmt"
	"log"
	"time"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func websocketHello(address string) {
	url := "ws://" + address + "/ws"

	creds, err := clientconfig.Load()
	if err != nil {
		log.Fatal("not paired yet, run pair command first")
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("websocket connection failed:", err)
	}
	defer conn.Close()

	authPayload, err := protocol.EncodePayload(protocol.AuthRequest{
		Token: creds.AuthToken,
	})
	if err != nil {
		log.Fatal("failed to encode auth payload:", err)
	}

	authMessage := protocol.Message{
		Type:      "auth",
		ID:        "auth_1",
		Timestamp: time.Now().Unix(),
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

		helloPayload, err := protocol.EncodePayload("hello from authenticated cli")
	if err != nil {
		log.Fatal("failed to encode hello payload:", err)
	}

	helloMessage := protocol.Message{
		Type:      "hello",
		ID:        "hello_1",
		Timestamp: time.Now().Unix(),
		Payload:   helloPayload,
	}

	err = conn.WriteJSON(helloMessage)
	if err != nil {
		log.Fatal("failed to send hello message:", err)
	}

	var helloResponse protocol.Message

	err = conn.ReadJSON(&helloResponse)
	if err != nil {
		log.Fatal("failed to read hello response:", err)
	}

	fmt.Println("Post-auth response:")
	fmt.Println("Type:", helloResponse.Type)
	fmt.Println("ID:", helloResponse.ID)
	fmt.Println("Payload:", string(helloResponse.Payload))
}