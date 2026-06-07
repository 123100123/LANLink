package ws

import (
	"fmt"
	"log"
	"strings"
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

func SendDirectMessage(address string, parts []string) {
	if len(parts) == 0 {
		log.Fatal("message text is required")
	}

	text := strings.Join(parts, " ")

	conn := ConnectAuthenticated(address)
	defer conn.Close()

	payload, err := protocol.EncodePayload(protocol.DirectMessagePayload{
		Text: text,
	})
	if err != nil {
		log.Fatal("failed to encode message payload:", err)
	}

	msg := protocol.Message{
		Type:      "direct_message",
		ID:        "msg_1",
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	err = conn.WriteJSON(msg)
	if err != nil {
		log.Fatal("failed to send message:", err)
	}

	var response protocol.Message

	err = conn.ReadJSON(&response)
	if err != nil {
		log.Fatal("failed to read response:", err)
	}

	err = conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"),
	)
	if err != nil {
		log.Println("failed to send close frame:", err)
	}

	if response.Type != "direct_message.response" {
		log.Fatal("unexpected response:", response.Type)
	}

	var responsePayload protocol.DirectMessageResponse

	err = protocol.DecodePayload(response.Payload, &responsePayload)
	if err != nil {
		log.Fatal("failed to decode response:", err)
	}

	fmt.Println("Message sent")
	fmt.Println("Agent response:", responsePayload.Status)
}
