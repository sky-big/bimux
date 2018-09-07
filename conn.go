package bimux

import (
	"github.com/gorilla/websocket"
)

// mockgen -source=./conn.go -destination=conn_mock.go -package=bimux
type Connection interface {
	Close()
	ReadPacket() (p []byte, err error)
	WritePacket(data []byte) error
}

type connection struct {
	con *websocket.Conn
}

func newWSConn(c *websocket.Conn) Connection {
	return &connection{con: c}
}

func (c *connection) Close() {
	c.con.Close()
}

func (c *connection) ReadPacket() (p []byte, err error) {
	_, message, err := c.con.ReadMessage()
	return message, err
}

func (c *connection) WritePacket(data []byte) error {

	return c.con.WriteMessage(websocket.BinaryMessage, data)
}
