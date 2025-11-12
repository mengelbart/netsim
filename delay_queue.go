package netsim

import (
	"container/heap"
	"time"
)

var _ heap.Interface = (*delayQueue)(nil)

type delayQueue struct {
	delay   time.Duration
	packets []*queuedPacket
}

func newDelayQueue(delay time.Duration) *delayQueue {
	return &delayQueue{
		delay:   delay,
		packets: []*queuedPacket{},
	}
}

func (q *delayQueue) push(pkt *queuedPacket) {
	pkt.due = time.Now().Add(q.delay)
	heap.Push(q, pkt)
}

func (q *delayQueue) pop() *queuedPacket {
	pkt := heap.Pop(q)
	return pkt.(*queuedPacket)
}

func (q *delayQueue) empty() bool {
	return len(q.packets) == 0
}

func (q *delayQueue) next() time.Time {
	if q.empty() {
		return time.Time{}
	}
	return q.packets[0].due
}

// Len implements heap.Interface.
func (q *delayQueue) Len() int {
	return len(q.packets)
}

// Less implements heap.Interface.
func (q *delayQueue) Less(i int, j int) bool {
	return q.packets[i].due.Before(q.packets[j].due)
}

// Pop implements heap.Interface.
func (q *delayQueue) Pop() any {
	n := len(q.packets)
	pkt := q.packets[n-1]
	q.packets = q.packets[0 : n-1]
	return pkt
}

// Push implements heap.Interface.
func (q *delayQueue) Push(x any) {
	pkt := x.(*queuedPacket)
	q.packets = append(q.packets, pkt)
}

// Swap implements heap.Interface.
func (q *delayQueue) Swap(i int, j int) {
	q.packets[i], q.packets[j] = q.packets[j], q.packets[i]
}
