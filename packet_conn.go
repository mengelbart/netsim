package netsim

import (
	"net"
	"time"
)

var _ net.PacketConn = (*packetConn)(nil)
var _ net.Conn = (*packetConn)(nil)

type packetConn struct {
	writer PacketWriter
}

func newPacketConn(w PacketWriter) *packetConn {
	return &packetConn{
		writer: w,
	}
}

// Read implements net.Conn.
func (p *packetConn) Read(b []byte) (n int, err error) {
	panic("unimplemented")
}

// RemoteAddr implements net.Conn.
func (p *packetConn) RemoteAddr() net.Addr {
	panic("unimplemented")
}

// Write implements net.Conn.
func (p *packetConn) Write(b []byte) (n int, err error) {
	return p.writer.WritePacket(b, Attributes{})
}

// Close implements net.PacketConn.
func (p *packetConn) Close() error {
	panic("unimplemented")
}

// LocalAddr implements net.PacketConn.
func (p *packetConn) LocalAddr() net.Addr {
	panic("unimplemented")
}

// ReadFrom implements net.PacketConn.
func (*packetConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	panic("unimplemented")
}

// SetDeadline implements net.PacketConn.
func (p *packetConn) SetDeadline(t time.Time) error {
	panic("unimplemented")
}

// SetReadDeadline implements net.PacketConn.
func (p *packetConn) SetReadDeadline(t time.Time) error {
	panic("unimplemented")
}

// SetWriteDeadline implements net.PacketConn.
func (p *packetConn) SetWriteDeadline(t time.Time) error {
	panic("unimplemented")
}

// WriteTo implements net.PacketConn.
func (*packetConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	panic("unimplemented")
}
