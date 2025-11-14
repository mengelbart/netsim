package netsim

import (
	"time"

	"golang.org/x/time/rate"
)

type RateQueue struct {
	limiter     *rate.Limiter
	packets     []*queuedPacket
	queueSize   int
	currentSize int
	headDrop    bool
}

func NewRateQueue(bitrate float64, burst int, queueSize int) *RateQueue {
	return &RateQueue{
		limiter:     rate.NewLimiter(rate.Limit(bitrate), burst),
		packets:     []*queuedPacket{},
		queueSize:   queueSize,
		currentSize: 0,
		headDrop:    false,
	}
}

func (q *RateQueue) push(pkt *queuedPacket) {
	if q.currentSize+len(pkt.payload) >= q.queueSize {
		if q.headDrop && len(q.packets) > 0 {
			q.packets = q.packets[1:]
			q.packets = append(q.packets, pkt)
		}
		return
	}
	q.packets = append(q.packets, pkt)
	q.currentSize += len(pkt.payload)
}

func (q *RateQueue) pop() (pkt *queuedPacket) {
	if q.empty() {
		return nil
	}
	if !q.limiter.AllowN(time.Now(), len(q.packets[0].payload)) {
		return nil
	}
	pkt, q.packets = q.packets[0], q.packets[1:]
	q.currentSize -= len(pkt.payload)
	return pkt
}

func (q *RateQueue) empty() bool {
	return len(q.packets) == 0
}

func (q *RateQueue) next() time.Time {
	if q.empty() {
		return time.Time{}
	}
	now := time.Now()
	if q.limiter.TokensAt(now) > float64(len(q.packets[0].payload)) {
		return now
	}
	res := q.limiter.ReserveN(now, len(q.packets[0].payload))
	delay := res.Delay()
	res.Cancel()
	return now.Add(delay)
}
