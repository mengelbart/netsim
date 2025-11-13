package netsim

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDelayQueueNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		qn := NewQueueNode(NewDelayQueue(time.Second))
		var w PacketWriter
		ch := make(chan packet, 1)
		w = PacketWriterFunc(func(b []byte, i PacketInfo) (int, error) {
			pkt := packet{
				payload: make([]byte, len(b)),
				info:    i,
			}
			n := copy(pkt.payload, b)
			ch <- pkt
			return n, nil
		})
		w = qn.Link(w)
		start := time.Now()
		payload := make([]byte, 1000)
		payload[17] = 0x17
		n, err := w.WritePacket(payload, PacketInfo{})
		pkt := <-ch
		assert.Equal(t, payload, pkt.payload)
		duration := time.Since(start)
		assert.NoError(t, err)
		assert.Equal(t, 1000, n)
		assert.GreaterOrEqual(t, duration.Milliseconds(), time.Second.Milliseconds())
		assert.NoError(t, qn.Close())
	})
}

func TestRateQueueNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		qn := NewQueueNode(NewRateQueue(10_000, 3000, 10))
		var w PacketWriter
		var bytes int
		start := time.Now()
		w = PacketWriterFunc(func(b []byte, i PacketInfo) (int, error) {
			bytes += len(b)
			return len(b), nil
		})
		w = qn.Link(w)

		for time.Since(start) < 5*time.Second {
			_, err := w.WritePacket(make([]byte, 1500), PacketInfo{})
			assert.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}
		end := time.Now()
		assert.NoError(t, qn.Close())

		duration := end.Sub(start)
		assert.Less(t, float64(bytes)/duration.Seconds(), 1.1*float64(10_000))
	})
}
