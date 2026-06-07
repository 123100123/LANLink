package ws

import (
	"time"

	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func handleHello(
	conn *websocket.Conn,
	msg protocol.Message,
) {

	payload, err := protocol.EncodePayload(
		"hello from authenticated lanlink agent",
	)

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

func writeError(
	conn *websocket.Conn,
	id string,
	reason string,
) {

	payload, err := protocol.EncodePayload(
		reason,
	)

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
