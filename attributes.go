package netsim

import "net"

type PacketInfo struct {
	Src net.Addr
	Dst net.Addr
	ECN ECN
}
