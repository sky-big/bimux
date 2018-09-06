package bimux

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type rpcServeFunc func(route uint32, req []byte) (rsp []byte)
type onewayFunc func(route uint32, req []byte)

var ErrTimeout error = errors.New("timeout")

type Muxer struct {
	conn       Connection
	writeMutex sync.Mutex
	readMutex  sync.Mutex

	callerRsp      map[uint64]chan *Message
	callerRspMutex sync.Mutex

	//	calledReqs      map[uintptr]*Message
	//	calledReqsMutex sync.Mutex
	rpcServeHook    rpcServeFunc
	onewayServeHook onewayFunc

	wg sync.WaitGroup

	number   uint64
	existErr error
}

func NewWebSocketMuxer(c *websocket.Conn, rpcServeHook rpcServeFunc, onewayServeHook onewayFunc) (*Muxer, error) {
	return newMuxer(newWSConn(c), rpcServeHook, onewayServeHook)
}

func newMuxer(conn Connection, rpcServeHook rpcServeFunc, onewayServeHook onewayFunc) (*Muxer, error) {
	m := &Muxer{
		conn:            conn,
		rpcServeHook:    rpcServeHook,
		onewayServeHook: onewayServeHook,
		callerRsp:       make(map[uint64]chan *Message),
	}
	m.wg.Add(1)
	go m.loop()
	return m, nil
}

func (m *Muxer) OnewaySend(route uint32, data []byte) {
	m.writeMsg(m.newMsg(Flag_oneway, route, data))
}

/*
for rpc : as role of client
*/
func (m *Muxer) Rpc(route uint32, req []byte, timeout time.Duration) (rsp []byte, err error) {
	// in case response will be handled before Ask get lock, create channel first and send request.
	reqMsg := m.newMsg(Flag_request, route, req)
	waitChan := m.register(reqMsg.Number)
	defer m.unRegister(reqMsg.Number)

	if err := m.writeMsg(reqMsg); err != nil {
		return nil, err
	}
	select {
	case answer := <-waitChan:
		return answer.Data, nil
	case <-time.After(timeout):
		return nil, ErrTimeout

	}
}

func (m *Muxer) Wait() error {
	m.wg.Wait()
	return m.existErr
}

func (m *Muxer) readMsg() (*Message, error) {
	m.readMutex.Lock()
	defer m.readMutex.Unlock()
	pack, err := m.conn.ReadPacket()
	if err != nil {
		return nil, err
	}
	var msg Message
	err = proto.Unmarshal(pack, &msg)
	return &msg, err
}

func (m *Muxer) newMsg(flag Flag, route uint32, data []byte) *Message {
	return &Message{
		Number: atomic.AddUint64(&m.number, uint64(1)),
		Flag:   flag,
		Route:  route,
		Data:   data,
	}
}

func (m *Muxer) responseMsg(reqMsg *Message, data []byte) *Message {
	return &Message{
		Number: reqMsg.Number,
		Flag:   Flag_response,
		Route:  reqMsg.Route,
		Data:   data,
	}
}

func (m *Muxer) writeMsg(msg *Message) error {
	m.writeMutex.Lock()
	defer m.writeMutex.Unlock()
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	err = m.conn.WritePacket(b)
	return err
}

func (m *Muxer) register(number uint64) chan *Message {
	waitChan := make(chan *Message, 1)
	m.callerRspMutex.Lock()
	m.callerRsp[number] = waitChan
	m.callerRspMutex.Unlock()
	return waitChan
}

func (m *Muxer) unRegister(number uint64) {
	m.callerRspMutex.Lock()
	close(m.callerRsp[number])
	delete(m.callerRsp, number)
	m.callerRspMutex.Unlock()
}

func (m *Muxer) notify(number uint64, msg *Message) {
	m.callerRspMutex.Lock()
	if waitChan, ok := m.callerRsp[number]; ok {
		waitChan <- msg
	}
	m.callerRspMutex.Unlock()
}

func (m *Muxer) loop() {
	defer m.wg.Done()
	for {
		msg, err := m.readMsg()
		if err != nil {
			return
		}
		switch msg.Flag {
		case Flag_response:
			m.notify(msg.Number, msg)
		case Flag_request:
			if m.rpcServeHook != nil {
				go func(tmp *Message) {
					rsp := m.rpcServeHook(tmp.Route, tmp.Data)
					rspMsg := m.responseMsg(tmp, rsp)
					m.writeMsg(rspMsg)
				}(msg)
			}
		case Flag_oneway:
			if m.onewayServeHook != nil {
				go func(tmp *Message) {
					m.onewayServeHook(tmp.Route, tmp.Data)
				}(msg)
			}
		}
	}
}
