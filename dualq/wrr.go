package dualq

type wrr struct {
	l4sWeight int

	count int
}

func newWrr() *wrr {
	return &wrr{
		l4sWeight: 15, // 15 L4S packets per 1 classic packet = 1/16 weight
	}
}

func (w *wrr) schedule(hasL4S, hasClassic bool) (scheduleL4S bool) {
	if !hasL4S && !hasClassic {
		panic("schedule called with no packets in queues")
	}

	if !hasL4S && hasClassic {
		// schedule classic immediately and restart round
		w.count = 0
		return false
	}

	if w.count < w.l4sWeight {
		// schedule L4S
		w.count++
		return true
	}

	// schedule classic -> round completed

	if !hasClassic {
		// no classic packet available -> schedule L4S in new round
		w.count = 1 // first l4s in new round
		return true
	}

	w.count = 0
	return false
}
