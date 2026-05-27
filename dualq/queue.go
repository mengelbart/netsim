package dualq

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/mengelbart/netsim"
)

type dualPi2 struct {
	k  float64
	cq *internalQueue
	lq *internalQueue

	MTU   int
	limit int

	mtx  sync.Mutex // protect p_CL and p_C
	p_CL float64
	p_C  float64

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
		thresholdLen: 1,                       // Th_len
		pLmax:        1,                       // p_Lmax
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

		if scheduleLq { //  L4S scheduled
			pkt := q.lq.pop()

			if q.p_CL < q.pLmax { // Check for overload saturation
				var p_prime_L float64
				if q.lq.len() > q.thresholdLen { // >1 packet queued
					p_prime_L = q.laqm() // Native LAQM
				} else {
					p_prime_L = 0 // Suppress marking 1 pkt queue
				}
				q.mtx.Lock()
				p_L := max(p_prime_L, q.p_CL) // Combining function
				q.mtx.Unlock()

				var mark bool
				q.lq.recurCount, mark = recur(q.lq.recurCount, p_L)
				if mark { // linear marking
					pkt.mark()
				}
			} else { // overload saturation
				var mark, drop bool

				q.mtx.Lock()
				q.lq.recurCount, drop = recur(q.lq.recurCount, q.p_C)
				q.mtx.Unlock()

				if drop { // probability p_C = p'^2
					continue // revert to Classic drop due to overload
				}

				q.mtx.Lock()
				q.lq.recurCount, mark = recur(q.lq.recurCount, q.p_C)
				q.mtx.Unlock()

				if mark { // probability p_CL = k * p'
					pkt.mark() // linear marking of remaining packets
				}
			}

			return pkt

		} else { // Classic scheduled
			pkt := q.cq.pop()
			var mark bool

			q.mtx.Lock()
			q.cq.recurCount, mark = recur(q.cq.recurCount, q.p_C)
			q.mtx.Unlock()

			if mark { // probability p_C = p'^2
				if pkt.info.ECN == netsim.ECNNonECT || q.p_C >= q.pCmax { // ECN field = not-ECT OR Overload disables ECN
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
	curq := max(q.cq.time(), q.lq.time()) // use greatest queuing time

	q.p_prime = q.p_prime + q.alpha*(curq.Seconds()-q.target.Seconds()) + q.beta*(curq.Seconds()-q.prevq.Seconds())

	q.mtx.Lock()
	q.p_CL = q.k * q.p_prime       // Coupled L4S prob = base prob * coupling factor
	q.p_C = math.Pow(q.p_prime, 2) // Classic prob = (base prob)^2
	q.mtx.Unlock()

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
