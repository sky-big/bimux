package bimux

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type RpcServeFunc func(route int32, req []byte, m Muxer) []byte
type OnewayFunc func(route int32, req []byte, m Muxer)
type StopFunc func(Muxer)

var ErrTimeout error = errors.New("timeout")

// mockgen -source=./mux.go -destination=mux_mock.go -package=bimux
type Muxer interface {
	Send(route int32, data []byte) error
	Rpc(route int32, req []byte, timeout time.Duration) (rsp []byte, err error)
	Wait() error
	Close()
}

type muxer struct {
	conn       Connection
	writeMutex sync.Mutex
	readMutex  sync.Mutex

	callerRsp      map[uint64]chan *Message
	callerRspMutex sync.Mutex

	rpcServeHook    RpcServeFunc
	onewayServeHook OnewayFunc
	stopHook        StopFunc

	isStop bool

	wg sync.WaitGroup

	number   uint64
	existErr error
}

/*
 Connection with web(server)
*/
var upgrader = websocket.Upgrader{} // use default options

func NewWebSocketMuxer(
	w http.ResponseWriter,
	r *http.Request,
	rpcServeHook RpcServeFunc,
	onewayServeHook OnewayFunc,
) (Muxer, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return newMuxer(newWSConn(c), rpcServeHook, onewayServeHook, nil)
}

/*
 Connection with websocket addr(client)
*/
func Dial(
	addr string,
	rpcServeHook RpcServeFunc,
	onewayServeHook OnewayFunc,
	stopHook StopFunc,
) (Muxer, error) {
	c, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		return nil, err
	}

	return newMuxer(newWSConn(c), rpcServeHook, onewayServeHook, stopHook)
}

/*
 Send Message
*/
func (m *muxer) Send(route int32, data []byte) error {
	return m.writeMsg(m.newMsg(Flag_oneway, route, data))
}

/*
 Rpc Send Message
*/
func (m *muxer) Rpc(route int32, req []byte, timeout time.Duration) (rsp []byte, err error) {
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

func (m *muxer) Wait() error {
	m.wg.Wait()
	return m.existErr
}

func (m *muxer) Close() {
	m.isStop = true
	m.conn.Close()
}

func newMuxer(conn Connection, rpcServeHook RpcServeFunc, onewayServeHook OnewayFunc, stopHook StopFunc) (Muxer, error) {
	m := &muxer{
		conn:      conn,
		callerRsp: make(map[uint64]chan *Message),

		rpcServeHook:    rpcServeHook,
		onewayServeHook: onewayServeHook,
		stopHook:        stopHook,
	}

	m.wg.Add(1)
	go m.loop()

	return m, nil
}

func (m *muxer) readMsg() (*Message, error) {
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

func (m *muxer) newMsg(flag Flag, route int32, data []byte) *Message {
	return &Message{
		Number: atomic.AddUint64(&m.number, uint64(1)),
		Flag:   flag,
		Route:  route,
		Data:   data,
	}
}

func (m *muxer) responseMsg(reqMsg *Message, route int32, data []byte) *Message {
	return &Message{
		Number: reqMsg.Number,
		Flag:   Flag_response,
		Route:  route,
		Data:   data,
	}
}

func (m *muxer) writeMsg(msg *Message) error {
	m.writeMutex.Lock()
	defer m.writeMutex.Unlock()
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	err = m.conn.WritePacket(b)
	return err
}

func (m *muxer) register(number uint64) chan *Message {
	waitChan := make(chan *Message, 1)
	m.callerRspMutex.Lock()
	m.callerRsp[number] = waitChan
	m.callerRspMutex.Unlock()
	return waitChan
}

func (m *muxer) unRegister(number uint64) {
	m.callerRspMutex.Lock()
	close(m.callerRsp[number])
	delete(m.callerRsp, number)
	m.callerRspMutex.Unlock()
}

func (m *muxer) notify(number uint64, msg *Message) {
	m.callerRspMutex.Lock()
	if waitChan, ok := m.callerRsp[number]; ok {
		waitChan <- msg
	}
	m.callerRspMutex.Unlock()
}

func (m *muxer) loop() {
	defer m.wg.Done()
	for {
		msg, err := m.readMsg()
		if err != nil {
			if !m.isStop && m.stopHook != nil {
				m.stopHook(m)
			}
			return
		}

		switch msg.Flag {
		case Flag_response:
			m.notify(msg.Number, msg)

		case Flag_request:
			if m.rpcServeHook != nil {
				go func(tmp *Message) {
					rsp := m.rpcServeHook(tmp.Route, tmp.Data, m)
					rspMsg := m.responseMsg(tmp, tmp.Route, rsp)
					m.writeMsg(rspMsg)
				}(msg)
			}

		case Flag_oneway:
			if m.onewayServeHook != nil {
				go func(tmp *Message) {
					m.onewayServeHook(tmp.Route, tmp.Data, m)
				}(msg)
			}
		}
	}
}
