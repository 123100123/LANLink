package ws

import (
	"fmt"
	"log"
	"time"

	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func sendHello(conn *websocket.Conn) {
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
