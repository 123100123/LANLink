package wsutil

import (
	"sync"

	"github.com/gorilla/websocket"
)

type SafeConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func NewSafeConn(conn *websocket.Conn) *SafeConn {
	return &SafeConn{
		conn: conn,
	}
}

func (c *SafeConn) ReadJSON(v any) error {
	return c.conn.ReadJSON(v)
}

func (c *SafeConn) WriteJSON(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.conn.WriteJSON(v)
}

func (c *SafeConn) Close() error {
	return c.conn.Close()
}

func (c *SafeConn) Raw() *websocket.Conn {
	return c.conn
}
