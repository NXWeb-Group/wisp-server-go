package types

import (
	"net"
	"sync"
)

type WispFrame struct {
	Type     PACKET_TYPE
	StreamID uint32
	Payload  []byte
}

type PACKET_TYPE uint8

const (
	CONNECT  PACKET_TYPE = 0x01
	DATA     PACKET_TYPE = 0x02
	CONTINUE PACKET_TYPE = 0x03
	CLOSE    PACKET_TYPE = 0x04
)

type Connect struct {
	Type STREAM_TYPE
	Port uint16
	Host []byte
}

type STREAM_TYPE uint8

const (
	TCP = 0x01
	UDP = 0x02
)

type Socket struct {
	TCP *net.TCPConn
	UDP *net.UDPConn
}

type Streams struct {
	Sockets map[uint32]Socket
	Mutex   sync.RWMutex
}
