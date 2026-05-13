package dualq

import "time"

type queuePacket struct {
	packet   *packet
	queuedTs time.Time
}

type ramp struct {
	queue []*queuePacket
	bytes int

	minThreshold time.Duration
	rangee       time.Duration
	thresholdLen int
	pLmax        float64

	maxThreshold time.Duration

	recurCount float64
}

func newRamp() *ramp {
	return &ramp{
		queue:        make([]*queuePacket, 0),
		minThreshold: 800 * time.Microsecond,
		rangee:       400 * time.Microsecond,
		thresholdLen: 1,
		pLmax:        1,
		// TODO: set maxThreshold (maxTH)
	}
}

func (r *ramp) push(pkt *packet) {
	qp := &queuePacket{
		packet:   pkt,
		queuedTs: time.Now(),
	}

	r.queue = append(r.queue, qp)
	r.bytes += len(pkt.payload)
}

func (p *ramp) pop() *packet {
	if len(p.queue) == 0 {
		return nil
	}

	pkt := p.queue[0]
	p.queue = p.queue[1:]
	p.bytes -= len(pkt.packet.payload)

	return pkt.packet
}

func (p *ramp) byt() int {
	return p.bytes
}

func (p *ramp) time() time.Duration {
	if len(p.queue) == 0 {
		return 0
	}

	return time.Since(p.queue[0].queuedTs)
}

func (p *ramp) laqm() float64 {
	qdelay := p.time()
	if qdelay >= p.maxThreshold {
		return 1
	} else if qdelay > p.minThreshold {
		return float64(qdelay) - float64(p.minThreshold)/float64(p.rangee)
	} else {
		return 0
	}
}
