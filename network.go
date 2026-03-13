package netsim

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"
)

type location bool

const (
	LeftLocation  location = true
	RightLocation location = false
)

type Net struct {
	leftNICs     map[netip.Addr]*NIC
	rightNICs    map[netip.Addr]*NIC
	forwardPath  *Path
	backwardPath *Path
	forward      PacketWriter
	backward     PacketWriter
}

func NewNet(forward, backward []Node) *Net {
	net := &Net{
		leftNICs:     map[netip.Addr]*NIC{},
		rightNICs:    map[netip.Addr]*NIC{},
		forwardPath:  NewPath(forward),
		backwardPath: NewPath(backward),
		forward:      nil,
		backward:     nil,
	}
	net.forward = net.forwardPath.Connect(net.packetWriter(net.rightNICs))
	net.backward = net.backwardPath.Connect(net.packetWriter(net.leftNICs))
	return net
}

func (n *Net) Close() error {
	return errors.Join(
		n.forwardPath.Close(),
		n.backwardPath.Close(),
	)
}

func (n *Net) NIC(loc location, address netip.Addr) *NIC {
	var nic *NIC
	switch loc {
	case LeftLocation:
		nic = NewNIC(address, n.forward)
		n.leftNICs[address] = nic
	case RightLocation:
		nic = NewNIC(address, n.backward)
		n.rightNICs[address] = nic
	}
	return nic
}

func (n *Net) packetWriter(table map[netip.Addr]*NIC) PacketWriter {
	return PacketWriterFunc(func(b []byte, a PacketInfo) (int, error) {
		var addr netip.Addr
		switch dst := a.Dst.(type) {
		// case *net.IPAddr:
		// case *net.IPNet:
		// case *net.TCPAddr:
		case *net.UDPAddr:
			addr = dst.AddrPort().Addr()
		// case *net.UnixAddr:
		default:
			return 0, fmt.Errorf("unexpected net.Addr: %#v", dst)
		}
		nic, ok := table[addr]
		if !ok {
			return 0, fmt.Errorf("unreachable: unknown address: %v", addr)
		}
		return nic.write(b, a)
	})
}

func (n *Net) WriteTcLogForwardPath(filePath string, duration time.Duration) error {
	now := time.Now()

	startTc := newTcLogEntry(n.forwardPath, now, duration)
	endTc := newTcLogEntry(n.forwardPath, now.Add(duration), 0)

	startBuf, err := startTc.MarshalJSON()
	if err != nil {
		return err
	}
	endBuf, err := endTc.MarshalJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, append(append(startBuf, '\n'), endBuf...), 0o644)
}
