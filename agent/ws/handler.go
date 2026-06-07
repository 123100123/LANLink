package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	conn, err := upgrader.Upgrade(
		w,
		r,
		nil,
	)

	if err != nil {
		log.Println(
			"websocket upgrade failed:",
			err,
		)
		return
	}

	defer conn.Close()

	log.Println(
		"websocket client connected",
	)

	device, ok := Authenticate(conn)

	if !ok {
		log.Println(
			"websocket authentication failed",
		)
		return
	}

	log.Println(
		"websocket authenticated:",
		device.DeviceID,
	)

	RunSession(
		conn,
		device,
	)
}
