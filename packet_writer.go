package netsim

type PacketWriter interface {
	WritePacket([]byte, Attributes) (int, error)
}

type PacketWriterFunc func([]byte, Attributes) (int, error)

func (f PacketWriterFunc) WritePacket(b []byte, a Attributes) (int, error) {
	return f(b, a)
}
