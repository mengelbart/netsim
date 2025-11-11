package loss

import "github.com/mengelbart/netsim"

type Writer struct {
}

func NewWriter() netsim.Node {
	return &Writer{}
}

func (l *Writer) Link(pw netsim.PacketWriter) netsim.PacketWriter {
	return netsim.PacketWriterFunc(func(b []byte, a netsim.Attributes) (int, error) {
		if false {
			return len(b), nil
		}
		return pw.WritePacket(b, a)
	})
}

func (l *Writer) Close() error {
	return nil
}
