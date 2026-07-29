package netsim

import (
	"math"
	"time"

	"golang.org/x/time/rate"
)

const (
	codelDefaultTarget   = 5 * time.Millisecond
	codelDefaultInterval = 100 * time.Millisecond
)

var _ queue = (*CodelQueue)(nil)

// CodelQueue is a queue that rate-limits egress traffic and applies the
// CoDel (Controlled Delay) active queue management algorithm from RFC 8289
// to decide which packets to drop under sustained congestion.
type CodelQueue struct {
	limiter  *rate.Limiter
	target   time.Duration
	interval time.Duration
	maxBytes int

	packets []*queuedPacket
	bytes   int

	// maxPacket tracks the largest packet ever seen and acts as CoDel's
	// stand-in for the path MTU when deciding if the queue is "shallow
	// enough" to skip dropping.
	maxPacket int

	dropping       bool
	firstAboveTime time.Time
	dropNext       time.Time
	count          int

	drops int
}

// NewCodelQueue creates a CodelQueue that forwards packets at bitrate bit/s
// (with the given burst, in bytes). maxBytes bounds the queue size in bytes
// (0 means unbounded, subject only to CoDel's sojourn-time drops). target
// and interval configure CoDel and default to 5ms/100ms when zero.
func NewCodelQueue(bitrate float64, burst int, maxBytes int, target, interval time.Duration) *CodelQueue {
	if target <= 0 {
		target = codelDefaultTarget
	}
	if interval <= 0 {
		interval = codelDefaultInterval
	}
	return &CodelQueue{
		limiter:  rate.NewLimiter(rate.Limit(bitrate/8), burst), // bit/s -> byte/s
		target:   target,
		interval: interval,
		maxBytes: maxBytes,
	}
}

// Drops returns the cumulative number of packets dropped by CoDel or
// because the queue was full.
func (q *CodelQueue) Drops() int {
	return q.drops
}

func (q *CodelQueue) push(pkt *queuedPacket) {
	n := len(pkt.payload)
	if n > q.maxPacket {
		q.maxPacket = n
	}
	if q.maxBytes > 0 && q.bytes+n > q.maxBytes {
		q.drops++
		return
	}
	pkt.due = time.Now() // repurposed here as this packet's enqueue time
	q.packets = append(q.packets, pkt)
	q.bytes += n
}

func (q *CodelQueue) empty() bool {
	return len(q.packets) == 0
}

func (q *CodelQueue) dequeue() *queuedPacket {
	if q.empty() {
		return nil
	}
	pkt := q.packets[0]
	q.packets = q.packets[1:]
	q.bytes -= len(pkt.payload)
	return pkt
}

// shouldDrop dequeues the head packet and reports whether its sojourn time
// makes it a candidate for dropping, per RFC 8289 section 5.
func (q *CodelQueue) shouldDrop(now time.Time) (pkt *queuedPacket, okToDrop bool) {
	pkt = q.dequeue()
	if pkt == nil {
		q.firstAboveTime = time.Time{}
		return nil, false
	}

	sojourn := now.Sub(pkt.due)
	if sojourn < q.target || q.bytes <= q.maxPacket {
		q.firstAboveTime = time.Time{}
		return pkt, false
	}

	if q.firstAboveTime.IsZero() {
		q.firstAboveTime = now.Add(q.interval)
	} else if !now.Before(q.firstAboveTime) {
		return pkt, true
	}
	return pkt, false
}

func (q *CodelQueue) controlLaw(t time.Time) time.Time {
	return t.Add(time.Duration(float64(q.interval) / math.Sqrt(float64(q.count))))
}

// dequeueWithCodel removes and returns the next packet to send, applying
// CoDel's drop logic (RFC 8289, Appendix A). now should be the time at
// which the link is ready to send a packet.
func (q *CodelQueue) dequeueWithCodel(now time.Time) *queuedPacket {
	pkt, okToDrop := q.shouldDrop(now)

	if q.dropping {
		if !okToDrop {
			q.dropping = false
		} else if !now.Before(q.dropNext) {
			for q.dropping && !now.Before(q.dropNext) {
				q.count++
				q.drops++
				pkt, okToDrop = q.shouldDrop(now)
				if !okToDrop {
					q.dropping = false
				} else {
					q.dropNext = q.controlLaw(q.dropNext)
				}
			}
		}
	} else if okToDrop {
		q.drops++
		pkt, okToDrop = q.shouldDrop(now)
		q.dropping = true
		if now.Sub(q.dropNext) < 16*q.interval && q.count > 2 {
			q.count -= 2
		} else {
			q.count = 1
		}
		q.dropNext = q.controlLaw(now)
	}

	return pkt
}

func (q *CodelQueue) pop() *queuedPacket {
	if q.empty() {
		return nil
	}
	now := time.Now()
	head := len(q.packets[0].payload)
	if !q.limiter.AllowN(now, head) {
		return nil
	}
	return q.dequeueWithCodel(now)
}

func (q *CodelQueue) next() time.Time {
	if q.empty() {
		return time.Time{}
	}
	now := time.Now()
	head := len(q.packets[0].payload)
	if q.limiter.TokensAt(now) >= float64(head) {
		return now
	}
	res := q.limiter.ReserveN(now, head)
	delay := res.Delay()
	res.Cancel()
	return now.Add(delay)
}
