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

	MTU   int
	limit int

	p_CL    float64
	p_C     float64
	prevq   time.Duration
	p_prime float64

	scheduler *wrr
}

func newDualPi2(maxLinkRate int) *dualPi2 {
	k := float64(2)
	q := &dualPi2{
		k:     k,
		cq:    newPi2(k),
		lq:    newRamp(),
		limit: maxLinkRate * 250,
		// TODO: set MTU
		scheduler: newWrr(),
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
			p_prime_L := q.lq.laqm()      // Native LAQM
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

func (q *dualPi2) update() {
	curq := q.cq.time() // use queuing time of first-in Classic packet

	q.p_prime = q.p_prime + q.cq.alpha*(float64(curq)-float64(q.cq.target)) + q.cq.beta*(float64(curq)-float64(q.prevq))
	q.p_CL = q.k * q.p_prime       // Coupled L4S prob = base prob * coupling factor
	q.p_C = math.Pow(q.p_prime, 2) // Classic prob = (base prob)^2
	q.prevq = curq
}

func recur(recurCount, likelyhood float64) (float64, bool) {
	recurCount += likelyhood

	if recurCount > 1 {
		recurCount--
		return recurCount, true
	}

	return recurCount, false
}
