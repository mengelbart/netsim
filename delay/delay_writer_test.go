package netsim

import (
	"maps"
	"testing"
	"testing/synctest"
	"time"

	"github.com/mengelbart/netsim"
	"github.com/stretchr/testify/assert"
)

func TestWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dw := NewWriter(time.Second)
		var w netsim.PacketWriter
		ch := make(chan packet, 1)
		w = netsim.PacketWriterFunc(func(b []byte, a netsim.Attributes) (int, error) {
			pkt := packet{
				payload:    make([]byte, len(b)),
				attributes: maps.Clone(a),
			}
			n := copy(pkt.payload, b)
			ch <- pkt
			return n, nil
		})
		w = dw.Link(w)
		start := time.Now()
		payload := make([]byte, 1000)
		payload[17] = 0x17
		n, err := w.WritePacket(payload, nil)
		pkt := <-ch
		assert.Equal(t, payload, pkt.payload)
		duration := time.Since(start)
		assert.NoError(t, err)
		assert.Equal(t, 1000, n)
		assert.GreaterOrEqual(t, duration.Milliseconds(), time.Second.Milliseconds())
		assert.NoError(t, dw.Close())
	})
}
