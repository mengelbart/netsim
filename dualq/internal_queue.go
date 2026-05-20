package dualq

import "time"

type queuePacket struct {
	packet   *packet
	queuedTs time.Time
}

type internalQueue struct {
	queue []*queuePacket
	bytes int

	recurCount float64
}

func newInternalQueue() *internalQueue {
	return &internalQueue{
		queue: make([]*queuePacket, 0),
	}
}

func (r *internalQueue) push(pkt *packet) {
	qp := &queuePacket{
		packet:   pkt,
		queuedTs: time.Now(),
	}

	r.queue = append(r.queue, qp)
	r.bytes += len(pkt.payload)
}

func (p *internalQueue) pop() *packet {
	if len(p.queue) == 0 {
		return nil
	}

	pkt := p.queue[0]
	p.queue = p.queue[1:]
	p.bytes -= len(pkt.packet.payload)

	return pkt.packet
}

func (p *internalQueue) byt() int {
	return p.bytes
}

func (p *internalQueue) time() time.Duration {
	if len(p.queue) == 0 {
		return 0
	}

	return time.Since(p.queue[0].queuedTs)
}
