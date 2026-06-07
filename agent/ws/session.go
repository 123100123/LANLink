package ws

import (
	"log"

	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func RunSession(
	conn *websocket.Conn,
	device *store.Device,
) {

	for {

		var msg protocol.Message

		err := conn.ReadJSON(
			&msg,
		)

		if err != nil {

			log.Println(
				"websocket read stopped for",
				device.DeviceID,
				":",
				err,
			)

			return
		}

		log.Println(
			"websocket message from",
			device.DeviceID,
			":",
			msg.Type,
		)

		switch msg.Type {

		case "hello":
			handleHello(
				conn,
				msg,
			)

		default:
			writeError(
				conn,
				msg.ID,
				"unknown message type",
			)
		}
	}
}
