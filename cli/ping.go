package main

import (
	"fmt"
	"log"
	"time"

	"github.com/123100123/lanlink/cli/ws"
	"github.com/123100123/lanlink/protocol"
)

func Run(address string) {
	conn := ws.ConnectAuthenticated(address)
	defer conn.Close()

	sentAt := time.Now().UnixMilli()

	payload, err := protocol.EncodePayload(
		protocol.PingPayload{
			SentAt: sentAt,
		},
	)

	if err != nil {
		log.Fatal("failed to encode ping payload:", err)
	}

	message := protocol.Message{
		Type:      "ping",
		ID:        "ping_1",
		Timestamp: sentAt,
		Payload:   payload,
	}

	err = conn.WriteJSON(message)
	if err != nil {
		log.Fatal("failed to send ping:", err)
	}

	var response protocol.Message

	err = conn.ReadJSON(&response)
	if err != nil {
		log.Fatal("failed to read pong:", err)
	}

	if response.Type != "pong" {
		log.Fatal("expected pong, got:", response.Type)
	}

	receivedAt := time.Now().UnixMilli()

	var pong protocol.PongPayload

	err = protocol.DecodePayload(response.Payload, &pong)
	if err != nil {
		log.Fatal("failed to decode pong payload:", err)
	}

	latency := receivedAt - pong.SentAt

	fmt.Println("Pong received")
	fmt.Println("Latency:", latency, "ms")
}