package bimux

import (
	"github.com/gorilla/websocket"
)

// mockgen -source=./conn.go -destination=conn_mock.go -package=mux
type Connection interface {
	ReadPacket() (p []byte, err error)
	WritePacket(data []byte) error
}

type WSConn struct {
	con *websocket.Conn
}

func newWSConn(c *websocket.Conn) *WSConn {
	return &WSConn{con: c}
}

func (c *WSConn) ReadPacket() (p []byte, err error) {

	_, message, err := c.con.ReadMessage()
	return message, err
}

func (c *WSConn) WritePacket(data []byte) error {

	return c.con.WriteMessage(websocket.BinaryMessage, data)
}
