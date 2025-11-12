package netsim

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
)

type NIC struct {
	address netip.Addr
	writer  PacketWriter
	conns   map[uint16]*packetConn
}

func NewNIC(address netip.Addr, writer PacketWriter) *NIC {
	return &NIC{
		address: address,
		writer:  writer,
		conns:   map[uint16]*packetConn{},
	}
}

func (n *NIC) ListenPacket(network string, address string) (*packetConn, error) {
	localAddrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return nil, err
	}
	if localAddrPort.Addr() != n.address {
		return nil, fmt.Errorf("invalid address: %v != %v", localAddrPort.Addr(), n.address)
	}
	if _, ok := n.conns[localAddrPort.Port()]; ok {
		return nil, errors.New("port already in use")
	}
	local := net.UDPAddrFromAddrPort(localAddrPort)
	pc := newPacketConn(n.writer, local, nil)
	n.conns[localAddrPort.Port()] = pc
	return pc, nil
}

func (n *NIC) Dial(network, address string) (*packetConn, error) {
	remoteAddrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return nil, err
	}
	var localPort uint16
	for _, ok := n.conns[localPort]; localPort < 1024 || ok; {
		localPort = uint16(rand.IntN(65535))
	}
	localAddrPort := netip.AddrPortFrom(n.address, localPort)
	pc := newPacketConn(n.writer, net.UDPAddrFromAddrPort(localAddrPort), net.UDPAddrFromAddrPort(remoteAddrPort))
	n.conns[localPort] = pc
	return pc, nil
}

func (n *NIC) write(b []byte, i PacketInfo) (int, error) {
	var port uint16
	switch dst := i.Dst.(type) {
	// case *net.IPAddr:
	// case *net.IPNet:
	// case *net.TCPAddr:
	case *net.UDPAddr:
		port = dst.AddrPort().Port()
	// case *net.UnixAddr:
	default:
		fmt.Errorf("unexpected net.Addr: %#v", dst)
	}
	conn, ok := n.conns[port]
	if !ok {
		return 0, fmt.Errorf("unreachable: unknown port: %v", port)
	}
	return conn.write(b, i), nil
}
