package netsim

import "net"

type packet struct {
	payload []byte
	info    PacketInfo
}

type PacketInfo struct {
	Src net.Addr
	Dst net.Addr
	ECN ECN
}

type PacketWriter interface {
	WritePacket([]byte, PacketInfo) (int, error)
}

type PacketWriterFunc func([]byte, PacketInfo) (int, error)

func (f PacketWriterFunc) WritePacket(b []byte, a PacketInfo) (int, error) {
	return f(b, a)
}
