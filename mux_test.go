package bimux

import (
	//	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewMsg(t *testing.T) {

	m := new(Muxer)
	m1 := m.newMsg(1, 1, nil)
	m2 := m.newMsg(2, 2, nil)
	assert.NotEqual(t, m1.Number, m2.Number)
	assert.Equal(t, m1.Number, uint64(1))
	assert.Equal(t, m2.Number, uint64(2))
}

func TestReadMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conn := NewMockConnection(ctrl)

	msg := &Message{Number: 1234}
	b, _ := proto.Marshal(msg)
	conn.EXPECT().ReadPacket().Return(b, nil)

	m := new(Muxer)
	m.conn = conn
	ret, err := m.readMsg()
	assert.Equal(t, msg.Number, ret.Number)
	assert.Equal(t, err, nil)

	msg = &Message{Number: 234}
	b, _ = proto.Marshal(msg)
	conn.EXPECT().ReadPacket().Return(b, nil)

	ret, err = m.readMsg()
	assert.Equal(t, msg.Number, ret.Number)
	assert.Equal(t, err, nil)
}

func TestWriteMsg(t *testing.T) {
}

func TestRegister(t *testing.T) {
	m := new(Muxer)
	m.callerRsp = make(map[uint64]chan *Message)
	var no uint64
	for no = 0; no < 100; no++ {
		ch := m.register(no)
		msg := &Message{Number: no}
		m.notify(no, msg)
		ret := <-ch
		assert.Equal(t, no, ret.Number)
	}
	assert.True(t, len(m.callerRsp) == 100)
	for no = 0; no < 100; no++ {
		m.unRegister(no)
	}
	assert.True(t, len(m.callerRsp) == 0)
}

func TestRpc(t *testing.T) {

}

type BenchCon struct {
	ch chan []byte
}

func newBenchCon() *BenchCon {
	return &BenchCon{make(chan []byte, 100000)}
}

func (c *BenchCon) ReadPacket() (p []byte, err error) {
	p = <-c.ch
	return p, nil
}
func (c *BenchCon) WritePacket(data []byte) error {
	var msg Message
	err := proto.Unmarshal(data, &msg)
	if err != nil {
		return err
	}
	msg.Flag = Flag_response
	ret, _ := proto.Marshal(&msg)
	c.ch <- ret
	return nil
}

func BenchmarkRegister(b *testing.B) {
	rpcCli, _ := newMuxer(newBenchCon(), nil, nil)
	//	b.RunParallel(func(pb *testing.PB) {
	//		for pb.Next() {
	//			rpcCli.Rpc(1, nil, time.Second)
	//		}
	//	})

	b.SetBytes(1000)
	for i := 0; i <= b.N; i++ {
		_, err := rpcCli.Rpc(1, nil, time.Second)
		if err != nil {
			b.FailNow()
		}
	}
}
