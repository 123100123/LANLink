package ws

import (
	"log"
	"time"

	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/internal/wsutil"
	"github.com/123100123/lanlink/protocol"
)

func handleHello(
	conn *wsutil.SafeConn,
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
	conn *wsutil.SafeConn,
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

func handlePing(
	conn *wsutil.SafeConn,
	msg protocol.Message,
) {
	var ping protocol.PingPayload

	err := protocol.DecodePayload(msg.Payload, &ping)
	if err != nil {
		writeError(conn, msg.ID, "invalid ping payload")
		return
	}

	payload, err := protocol.EncodePayload(
		protocol.PongPayload{
			SentAt:     ping.SentAt,
			ReceivedAt: time.Now().UnixMilli(),
		},
	)

	if err != nil {
		writeError(conn, msg.ID, "failed to encode pong payload")
		return
	}

	response := protocol.Message{
		Type:      "pong",
		ID:        msg.ID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	conn.WriteJSON(response)
}

func handleDirectMessage(
	conn *wsutil.SafeConn,
	msg protocol.Message,
	device *store.Device,
) {
	var payload protocol.DirectMessagePayload

	err := protocol.DecodePayload(msg.Payload, &payload)
	if err != nil {
		writeError(conn, msg.ID, "invalid direct message payload")
		return
	}

	log.Println("direct message from", device.DeviceID+":", payload.Text)

	responsePayload, err := protocol.EncodePayload(
		protocol.DirectMessageResponse{
			Status: "received",
		},
	)
	if err != nil {
		writeError(conn, msg.ID, "failed to encode direct message response")
		return
	}

	response := protocol.Message{
		Type:      "direct_message.response",
		ID:        msg.ID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   responsePayload,
	}

	conn.WriteJSON(response)
}
