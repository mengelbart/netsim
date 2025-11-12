package netsim

import (
	"errors"
	"net"
	"slices"
	"sync"
	"time"
)

var _ net.PacketConn = (*packetConn)(nil)
var _ net.Conn = (*packetConn)(nil)

type packetConn struct {
	lock sync.Mutex

	writer PacketWriter
	queue  chan packet

	localAddr  net.Addr
	remoteAddr net.Addr

	readDeadline    time.Time
	writeDeadline   time.Time
	deadlineUpdated chan struct{}

	closed   bool
	closedCh chan struct{}
}

func newPacketConn(w PacketWriter, local, remote net.Addr) *packetConn {
	return &packetConn{
		lock:            sync.Mutex{},
		writer:          w,
		queue:           make(chan packet, 1000),
		localAddr:       local,
		remoteAddr:      remote,
		readDeadline:    time.Time{},
		deadlineUpdated: make(chan struct{}),
	}
}

func (p *packetConn) write(b []byte, i PacketInfo) int {
	buf := make([]byte, len(b))
	n := copy(buf, b)
	select {
	case p.queue <- packet{
		payload: buf,
		info:    i,
	}:
	default:
	}
	return n
}

// Read implements net.Conn.
func (p *packetConn) Read(b []byte) (int, error) {
	n, _, err := p.ReadFrom(b)
	return n, err
}

// RemoteAddr implements net.Conn.
func (p *packetConn) RemoteAddr() net.Addr {
	return p.remoteAddr
}

// Write implements net.Conn.
func (p *packetConn) Write(b []byte) (n int, err error) {
	return p.WriteTo(b, p.remoteAddr)
}

// Close implements net.PacketConn.
func (p *packetConn) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	close(p.closedCh)
	return nil
}

// LocalAddr implements net.PacketConn.
func (p *packetConn) LocalAddr() net.Addr {
	return p.localAddr
}

// ReadFrom implements net.PacketConn.
func (p *packetConn) ReadFrom(b []byte) (int, net.Addr, error) {
	p.lock.Lock()
	if p.closed {
		p.lock.Unlock()
		return 0, nil, net.ErrClosed
	}
	deadline := p.readDeadline
	p.lock.Unlock()

	var pkt packet
	var timer <-chan time.Time
	if !deadline.IsZero() {
		timer = time.After(time.Until(deadline))
	}

	select {
	case pkt = <-p.queue:
	case <-p.closedCh:
		return 0, nil, net.ErrClosed
	case <-p.deadlineUpdated:
		return p.ReadFrom(b)
	case <-timer:
		return 0, nil, errors.New("read timeout")
	}
	n := copy(b, pkt.payload)
	return n, pkt.info.Src, nil
}

// SetDeadline implements net.PacketConn.
func (p *packetConn) SetDeadline(t time.Time) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.readDeadline = t
	p.writeDeadline = t
	select {
	case p.deadlineUpdated <- struct{}{}:
	default:
	}
	return nil
}

// SetReadDeadline implements net.PacketConn.
func (p *packetConn) SetReadDeadline(t time.Time) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.readDeadline = t
	select {
	case p.deadlineUpdated <- struct{}{}:
	default:
	}
	return nil
}

// SetWriteDeadline implements net.PacketConn.
func (p *packetConn) SetWriteDeadline(t time.Time) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.writeDeadline = t
	return nil
}

// WriteTo implements net.PacketConn.
func (p *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	p.lock.Lock()
	if p.closed {
		p.lock.Unlock()
		return 0, net.ErrClosed
	}
	deadline := p.writeDeadline
	p.lock.Unlock()

	if !deadline.IsZero() && time.Now().After(deadline) {
		return 0, errors.New("write timeout")
	}

	return p.writer.WritePacket(slices.Clone(b), PacketInfo{
		Src: p.localAddr,
		Dst: addr,
		ECN: 0,
	})
}
