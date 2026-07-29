package netsim

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCodelQueueNoDropUnderTarget(t *testing.T) {
	q := NewCodelQueue(0, 0, 0, 5*time.Millisecond, 100*time.Millisecond)
	start := time.Now()

	for i := range 20 {
		now := start.Add(time.Duration(i) * time.Millisecond)
		pkt := &queuedPacket{payload: make([]byte, 1000)}
		pkt.due = now
		q.packets = append(q.packets, pkt)
		q.bytes += len(pkt.payload)

		out := q.dequeueWithCodel(now.Add(2 * time.Millisecond))
		assert.NotNil(t, out)
	}
	assert.Equal(t, 0, q.Drops())
}

func TestCodelQueueDropsAfterSustainedDelay(t *testing.T) {
	q := NewCodelQueue(0, 0, 0, 5*time.Millisecond, 100*time.Millisecond)
	start := time.Now()

	for range 200 {
		pkt := &queuedPacket{payload: make([]byte, 1000)}
		pkt.due = start
		q.packets = append(q.packets, pkt)
		q.bytes += len(pkt.payload)
	}

	sent := 0
	for i := range 200 {
		now := start.Add(time.Duration(i+1) * time.Millisecond)
		if out := q.dequeueWithCodel(now); out != nil {
			sent++
		}
	}

	assert.Greater(t, q.Drops(), 0, "expected CoDel to drop packets under sustained queueing delay")
	assert.Greater(t, sent, 0)
}

func TestCodelQueueMaxBytesTailDrop(t *testing.T) {
	q := NewCodelQueue(0, 0, 1500, 5*time.Millisecond, 100*time.Millisecond)

	pkt1 := &queuedPacket{payload: make([]byte, 1000)}
	pkt2 := &queuedPacket{payload: make([]byte, 1000)}
	q.push(pkt1)
	q.push(pkt2)

	assert.Equal(t, 1, len(q.packets))
	assert.Equal(t, 1, q.Drops())
	assert.Equal(t, 1000, q.bytes)
}

func TestCodelQueueEmptyPop(t *testing.T) {
	q := NewCodelQueue(0, 0, 0, 0, 0)
	assert.Nil(t, q.pop())
	assert.True(t, q.empty())
}

func TestCodelQueueNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewCodelQueue(10_000, 1500, 10_000, 5*time.Millisecond, 100*time.Millisecond)
		qn := NewQueueNode(q)

		var w PacketWriter
		var bytes int
		start := time.Now()
		w = PacketWriterFunc(func(b []byte, i PacketInfo) (int, error) {
			bytes += len(b)
			return len(b), nil
		})
		w = qn.Link(w)

		for time.Since(start) < 20*time.Second {
			_, err := w.WritePacket(make([]byte, 1500), PacketInfo{})
			assert.NoError(t, err)
			time.Sleep(time.Millisecond)
		}
		synctest.Wait()
		end := time.Now()
		assert.NoError(t, qn.Close())

		duration := end.Sub(start)
		assert.Less(t, float64(bytes)/duration.Seconds(), 1.1*float64(10_000/8))
		assert.Greater(t, q.Drops(), 0, "expected CoDel to drop packets when arrival outpaces the link rate")
	})
}
