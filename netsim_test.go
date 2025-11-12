package netsim

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetSim(t *testing.T) {
	net := NewNet([]Node{}, []Node{})
	leftNic := net.NIC(LeftLocation, netip.MustParseAddr("127.0.0.1"))
	rightNic := net.NIC(RightLocation, netip.MustParseAddr("127.0.0.1"))
	leftConn, err := leftNic.Dial("udp", "127.0.0.1:8080")
	assert.NoError(t, err)
	rightConn, err := rightNic.ListenPacket("udp", "127.0.0.1:8080")
	assert.NoError(t, err)

	data := []byte("hello world")
	n, err := leftConn.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, n, len(data))

	buf := make([]byte, 1500)
	m, err := rightConn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, m, n)
}
