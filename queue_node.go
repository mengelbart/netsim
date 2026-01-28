package netsim

import (
	"errors"
	"slices"
	"sync"
	"time"
)

type queuedPacket struct {
	payload []byte
	info    PacketInfo
	writer  PacketWriter
	due     time.Time
}

type queue interface {
	push(*queuedPacket)
	pop() *queuedPacket
	empty() bool
	next() time.Time
}

type QueueNode struct {
	lock    sync.Mutex
	queue   queue
	packets chan *queuedPacket
	close   chan struct{}
	closed  bool
	wg      sync.WaitGroup
}

func NewQueueNode(q queue) *QueueNode {
	qn := &QueueNode{
		lock:    sync.Mutex{},
		queue:   q,
		packets: make(chan *queuedPacket),
		close:   make(chan struct{}),
		closed:  false,
		wg:      sync.WaitGroup{},
	}
	qn.wg.Go(qn.run)
	return qn
}

func (n *QueueNode) Link(pw PacketWriter) PacketWriter {
	return PacketWriterFunc(func(b []byte, pi PacketInfo) (int, error) {
		select {
		case n.packets <- &queuedPacket{
			payload: slices.Clone(b),
			info:    pi,
			writer:  pw,
		}:
		case <-n.close:
			return 0, errors.New("node closed")
		}

		return len(b), nil
	})
}

func (n *QueueNode) run() {
	for {
		if !n.schedule() {
			return
		}
	}
}

func (n *QueueNode) schedule() bool {
	n.lock.Lock()
	if n.closed {
		n.lock.Unlock()
		return false
	}
	n.lock.Unlock()
	var timer <-chan time.Time

	if !n.queue.empty() {
		timer = time.After(time.Until(n.queue.next()))
	}

	select {
	case pkt := <-n.packets:
		n.queue.push(pkt)
	case <-timer:
		pkt := n.queue.pop()
		if pkt != nil {
			_, _ = pkt.writer.WritePacket(pkt.payload, pkt.info)
		}

	case <-n.close:
		return false
	}
	return true
}

func (n *QueueNode) Close() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	defer n.wg.Wait()
	if n.closed {
		return nil
	}
	n.closed = true
	close(n.close)
	return nil
}
