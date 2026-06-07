package ws

import (
	"log"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/gorilla/websocket"
)

func Connect(address string) {
	conn := ConnectAuthenticated(address)
	defer conn.Close()

	sendHello(conn)
}

func ConnectAuthenticated(address string) *websocket.Conn {
	url := "ws://" + address + "/ws"

	creds, err := clientconfig.Load()
	if err != nil {
		log.Fatal("not paired yet, run pair command first")
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("websocket connection failed:", err)
	}

	authenticate(conn, creds.AuthToken)

	return conn
}
