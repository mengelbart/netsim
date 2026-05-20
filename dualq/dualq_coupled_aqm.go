package dualq

import (
	"context"
	"sync"
	"time"

	"github.com/mengelbart/netsim"
)

type packet struct {
	payload []byte
	info    netsim.PacketInfo
	writer  netsim.PacketWriter
}

func (p *packet) ecn() bool {
	return false // TODO
}

func (p *packet) mark() {
	// TODO
}

type Writer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	queue     chan *packet
	wg        sync.WaitGroup
}

func NewWriter() netsim.Node {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Writer{
		ctx:       ctx,
		cancelCtx: cancel,
		queue:     make(chan *packet, 1),
		wg:        sync.WaitGroup{},
	}
	w.wg.Go(w.run)
	return w
}

func (w *Writer) Link(pw netsim.PacketWriter) netsim.PacketWriter {
	return netsim.PacketWriterFunc(func(b []byte, a netsim.PacketInfo) (int, error) {
		pkt := &packet{
			payload: make([]byte, len(b)),
			info:    a,
			writer:  pw,
		}
		n := copy(pkt.payload, b)
		select {
		case w.queue <- pkt:
		default:
			// Drop packet if queue is busy
		}
		return n, nil
	})
}

func (w *Writer) run() {
	maxLinkRate := 1_000_000 // TODO: set max link rate correctly
	queue := newDualPi2(maxLinkRate)

	w.wg.Go(func() {
		queue.RunUpdates(w.ctx)
	})

	timer := time.NewTimer(0)
	for {
		select {
		case pkt := <-w.queue:
			queue.push(pkt)
		case <-timer.C:
		}
	}
}

// Close implements netsim.Node.
func (w *Writer) Close() error {
	defer w.wg.Done()
	w.cancelCtx()
	return nil
}
