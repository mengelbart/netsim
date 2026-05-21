package dualq

import (
	"context"
	"math"
	"time"

	"github.com/mengelbart/netsim"
)

type dualPi2 struct {
	k  float64
	cq *internalQueue
	lq *internalQueue

	MTU   int
	limit int

	p_CL    float64
	p_C     float64
	prevq   time.Duration
	p_prime float64

	scheduler *wrr

	// ramp values
	minThreshold time.Duration
	rangee       time.Duration
	thresholdLen int
	pLmax        float64
	maxThreshold time.Duration

	// pi2 values
	target  time.Duration
	rttMax  time.Duration
	pCmax   float64
	tUpdate time.Duration
	alpha   float64
	beta    float64
}

func newDualPi2(maxLinkRate int) *dualPi2 {
	k := float64(2)

	target := 15 * time.Millisecond
	rttMax := 100 * time.Millisecond
	tUpdate := min(target, rttMax/3.0)

	q := &dualPi2{
		k:     k,
		cq:    newInternalQueue(),
		lq:    newInternalQueue(),
		limit: maxLinkRate * 250,
		// TODO: set MTU
		scheduler: newWrr(),

		// ramp
		minThreshold: 800 * time.Microsecond,
		rangee:       400 * time.Microsecond,
		thresholdLen: 1,
		pLmax:        1,
		maxThreshold: 1200 * time.Microsecond, // TODO: maxTh not defined in papger

		// pi2
		target:  target,
		rttMax:  rttMax,
		pCmax:   min(1.0/math.Sqrt(k), 1.0),
		tUpdate: tUpdate,
		alpha:   0.1 * tUpdate.Seconds() / math.Sqrt(rttMax.Seconds()),
		beta:    0.3 / rttMax.Seconds(),
	}

	return q
}

func (q *dualPi2) push(pkt *packet) {
	if q.lq.byt()+q.cq.byt()+q.MTU > q.limit {
		// drop
		return
	}

	//  timestamp(pkt)     % only needed if using the sojourn technique

	if pkt.info.ECN == netsim.ECNECT0 || pkt.info.ECN == netsim.ECNCE {
		q.lq.push(pkt)
	} else {
		q.cq.push(pkt)
	}
}

// pop should be repeatedly called whenever the lower layer is ready to forward a packet
func (q *dualPi2) pop() *packet {
	for q.lq.byt()+q.cq.byt() > 0 {
		hasL4S := q.lq.byt() > 0
		hasClassic := q.cq.byt() > 0
		scheduleLq := q.scheduler.schedule(hasL4S, hasClassic)

		if scheduleLq {

			pkt := q.lq.pop()
			p_prime_L := q.laqm()         // Native LAQM
			p_L := max(p_prime_L, q.p_CL) // Combining function

			var mark bool
			q.lq.recurCount, mark = recur(q.lq.recurCount, p_L)
			if mark { // linear marking
				pkt.mark()
			}

			return pkt

		} else {
			pkt := q.cq.pop()

			var mark bool
			q.cq.recurCount, mark = recur(q.cq.recurCount, q.p_C)
			if mark { // probability p_C = p'^2
				if pkt.info.ECN == netsim.ECNNonECT { // if ECN field = not-ECT
					// drop packet
					continue
				}
				pkt.mark() // squared mark
			}

			return pkt
		}
	}

	return nil
}

func (q *dualPi2) empty() bool {
	return q.lq.byt()+q.cq.byt() == 0
}

func (q *dualPi2) update() {
	curq := q.cq.time() // use queuing time of first-in Classic packet

	q.p_prime = q.p_prime + q.alpha*(curq.Seconds()-q.target.Seconds()) + q.beta*(curq.Seconds()-q.prevq.Seconds())
	q.p_CL = q.k * q.p_prime       // Coupled L4S prob = base prob * coupling factor
	q.p_C = math.Pow(q.p_prime, 2) // Classic prob = (base prob)^2
	q.prevq = curq
}

func (q *dualPi2) RunUpdates(ctx context.Context) {
	ticker := time.NewTicker(q.tUpdate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		q.update()
	}
}

// laqm is Native L4S AQM
func (p *dualPi2) laqm() float64 {
	qdelay := p.lq.time()
	if qdelay >= p.maxThreshold {
		return 1
	} else if qdelay > p.minThreshold {
		return (qdelay.Seconds() - p.minThreshold.Seconds()) / p.rangee.Seconds()
	} else {
		return 0
	}
}

// recur returns true with a certain likelihood
func recur(recurCount, likelyhood float64) (float64, bool) {
	recurCount += likelyhood

	if recurCount > 1 {
		recurCount--
		return recurCount, true
	}

	return recurCount, false
}
