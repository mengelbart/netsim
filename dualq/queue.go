package dualq

import (
	"math"
	"time"

	"github.com/mengelbart/netsim"
)

type dualPi2 struct {
	k  float64
	cq *pi2
	lq *ramp
}

func newDualPi2() *dualPi2 {
	k := float64(2)
	q := &dualPi2{
		k:  k,
		cq: newPi2(k),
		lq: newRamp(),
	}
	return q
}

func (q *dualPi2) push(pkt *packet) {
	if pkt.info.ECN == netsim.ECNECT0 || pkt.info.ECN == netsim.ECNCE {
		q.lq.push(pkt)
	} else {
		q.cq.push(pkt)
	}
}

func (q *dualPi2) pop() *packet {
	return nil
}

type pi2 struct {
	target  time.Duration
	rttMax  time.Duration
	pCmax   float64
	tUpdate time.Duration
	alpha   float64
	beta    float64
}

func newPi2(k float64) *pi2 {
	target := 15 * time.Millisecond
	rttMax := 100 * time.Millisecond
	tUpdate := min(target, rttMax/3.0)
	return &pi2{
		target:  target,
		rttMax:  rttMax,
		pCmax:   min(1.0/math.Sqrt(k), 1.0),
		tUpdate: tUpdate,
		alpha:   0.1 * tUpdate.Seconds() / math.Sqrt(float64(rttMax)),
		beta:    0.3 / rttMax.Seconds(),
	}
}

func (p *pi2) push(pkt *packet) {

}

type ramp struct {
	minThreshold time.Duration
	rangee       time.Duration
	thresholdLen int
	pLmax        float64
}

func newRamp() *ramp {
	return &ramp{
		minThreshold: 800 * time.Microsecond,
		rangee:       400 * time.Microsecond,
		thresholdLen: 1,
		pLmax:        1,
	}
}

func (r *ramp) push(pkt *packet) {

}
