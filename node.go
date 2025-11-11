package netsim

import "io"

type Node interface {
	Link(PacketWriter) PacketWriter
	io.Closer
}
