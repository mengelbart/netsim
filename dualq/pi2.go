package dualq

import (
	"math"
	"time"
)

type pi2 struct {
	queue []*queuePacket
	bytes int

	target  time.Duration
	rttMax  time.Duration
	pCmax   float64
	tUpdate time.Duration
	alpha   float64
	beta    float64

	recurCount float64
}

func newPi2(k float64) *pi2 {
	target := 15 * time.Millisecond
	rttMax := 100 * time.Millisecond
	tUpdate := min(target, rttMax/3.0)
	return &pi2{
		queue:   make([]*queuePacket, 0),
		target:  target,
		rttMax:  rttMax,
		pCmax:   min(1.0/math.Sqrt(k), 1.0),
		tUpdate: tUpdate,
		alpha:   0.1 * tUpdate.Seconds() / math.Sqrt(float64(rttMax)),
		beta:    0.3 / rttMax.Seconds(),
	}
}

func (p *pi2) time() time.Duration {
	//  how long the head packet was in the Classic queue (cq).
	// The function cq.time() (not shown) subtracts the time stamped at enqueue from the current time
	// (see Note a below) and implicitly takes the current queuing delay as 0 if the queue is empty.

	if len(p.queue) == 0 {
		return 0
	}

	return time.Since(p.queue[0].queuedTs)
}

func (p *pi2) push(pkt *packet) {
	qp := &queuePacket{
		packet:   pkt,
		queuedTs: time.Now(),
	}

	p.queue = append(p.queue, qp)
	p.bytes += len(pkt.payload)
}

func (p *pi2) pop() *packet {
	if len(p.queue) == 0 {
		return nil
	}

	pkt := p.queue[0]
	p.queue = p.queue[1:]
	p.bytes -= len(pkt.packet.payload)

	return pkt.packet
}

func (p *pi2) byt() int {
	return p.bytes
}
