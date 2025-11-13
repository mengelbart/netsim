package netsim

import (
	"container/heap"
	"time"
)

var _ heap.Interface = (*DelayQueue)(nil)

type DelayQueue struct {
	delay   time.Duration
	packets []*queuedPacket
}

func NewDelayQueue(delay time.Duration) *DelayQueue {
	return &DelayQueue{
		delay:   delay,
		packets: []*queuedPacket{},
	}
}

func (q *DelayQueue) push(pkt *queuedPacket) {
	pkt.due = time.Now().Add(q.delay)
	heap.Push(q, pkt)
}

func (q *DelayQueue) pop() *queuedPacket {
	pkt := heap.Pop(q)
	return pkt.(*queuedPacket)
}

func (q *DelayQueue) empty() bool {
	return len(q.packets) == 0
}

func (q *DelayQueue) next() time.Time {
	if q.empty() {
		return time.Time{}
	}
	return q.packets[0].due
}

// Len implements heap.Interface.
func (q *DelayQueue) Len() int {
	return len(q.packets)
}

// Less implements heap.Interface.
func (q *DelayQueue) Less(i int, j int) bool {
	return q.packets[i].due.Before(q.packets[j].due)
}

// Pop implements heap.Interface.
func (q *DelayQueue) Pop() any {
	n := len(q.packets)
	pkt := q.packets[n-1]
	q.packets = q.packets[0 : n-1]
	return pkt
}

// Push implements heap.Interface.
func (q *DelayQueue) Push(x any) {
	pkt := x.(*queuedPacket)
	q.packets = append(q.packets, pkt)
}

// Swap implements heap.Interface.
func (q *DelayQueue) Swap(i int, j int) {
	q.packets[i], q.packets[j] = q.packets[j], q.packets[i]
}
