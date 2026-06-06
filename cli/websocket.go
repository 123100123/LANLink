package main

import (
	"fmt"
	"log"
	"time"

	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func websocketHello(address string) {
	url := "ws://" + address + "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("websocket connection failed:", err)
	}
	defer conn.Close()

	request := protocol.Message{
		Type:      "hello",
		ID:        "hello_1",
		Timestamp: time.Now().Unix(),
		Payload:   "hello from lanlink cli",
	}

	err = conn.WriteJSON(request)
	if err != nil {
		log.Fatal("failed to send websocket message:", err)
	}

	var response protocol.Message

	err = conn.ReadJSON(&response)
	if err != nil {
		log.Fatal("failed to read websocket response:", err)
	}

	fmt.Println("WebSocket response received")
	fmt.Println("Type:", response.Type)
	fmt.Println("ID:", response.ID)
	fmt.Println("Payload:", response.Payload)
}