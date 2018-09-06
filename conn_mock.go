// Code generated by MockGen. DO NOT EDIT.
// Source: ./conn.go

// Package bimux is a generated GoMock package.
package bimux

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockConnection is a mock of Connection interface
type MockConnection struct {
	ctrl     *gomock.Controller
	recorder *MockConnectionMockRecorder
}

// MockConnectionMockRecorder is the mock recorder for MockConnection
type MockConnectionMockRecorder struct {
	mock *MockConnection
}

// NewMockConnection creates a new mock instance
func NewMockConnection(ctrl *gomock.Controller) *MockConnection {
	mock := &MockConnection{ctrl: ctrl}
	mock.recorder = &MockConnectionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockConnection) EXPECT() *MockConnectionMockRecorder {
	return m.recorder
}

// ReadPacket mocks base method
func (m *MockConnection) ReadPacket() ([]byte, error) {
	ret := m.ctrl.Call(m, "ReadPacket")
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadPacket indicates an expected call of ReadPacket
func (mr *MockConnectionMockRecorder) ReadPacket() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadPacket", reflect.TypeOf((*MockConnection)(nil).ReadPacket))
}

// WritePacket mocks base method
func (m *MockConnection) WritePacket(data []byte) error {
	ret := m.ctrl.Call(m, "WritePacket", data)
	ret0, _ := ret[0].(error)
	return ret0
}

// WritePacket indicates an expected call of WritePacket
func (mr *MockConnectionMockRecorder) WritePacket(data interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WritePacket", reflect.TypeOf((*MockConnection)(nil).WritePacket), data)
}
