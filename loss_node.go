package netsim

import "math/rand/v2"

type LossNode struct {
	chance float64
}

func NewLossNode(chance float64) *LossNode {
	return &LossNode{
		chance: chance,
	}
}

func (n *LossNode) Link(pw PacketWriter) PacketWriter {
	return PacketWriterFunc(func(b []byte, pi PacketInfo) (int, error) {
		x := rand.Float64()
		if x < n.chance {
			return len(b), nil
		}
		return pw.WritePacket(b, pi)
	})
}

func (n *LossNode) Close() error {
	return nil
}
