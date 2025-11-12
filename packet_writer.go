package netsim

type PacketWriter interface {
	WritePacket([]byte, PacketInfo) (int, error)
}

type PacketWriterFunc func([]byte, PacketInfo) (int, error)

func (f PacketWriterFunc) WritePacket(b []byte, a PacketInfo) (int, error) {
	return f(b, a)
}
