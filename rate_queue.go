package netsim

import (
	"time"

	"golang.org/x/time/rate"
)

type rateQueue struct {
	limiter   *rate.Limiter
	packets   []*queuedPacket
	queueSize int
}

func newRateQueue(bitrate float64, burst int, queueSize int) *rateQueue {
	return &rateQueue{
		limiter:   rate.NewLimiter(rate.Limit(bitrate), burst),
		packets:   []*queuedPacket{},
		queueSize: queueSize,
	}
}

func (q *rateQueue) push(pkt *queuedPacket) {
	if len(q.packets) >= q.queueSize {
		return
	}
	q.packets = append(q.packets, pkt)
}

func (q *rateQueue) pop() (pkt *queuedPacket) {
	if q.empty() {
		return nil
	}
	if !q.limiter.AllowN(time.Now(), len(q.packets[0].payload)) {
		return nil
	}
	pkt, q.packets = q.packets[0], q.packets[1:]
	return pkt
}

func (q *rateQueue) empty() bool {
	return len(q.packets) == 0
}

func (q *rateQueue) next() time.Time {
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
