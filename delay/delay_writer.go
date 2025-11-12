package netsim

import (
	"context"
	"sync"
	"time"

	"github.com/mengelbart/netsim"
)

type packet struct {
	payload  []byte
	info     netsim.PacketInfo
	writer   netsim.PacketWriter
	deadline time.Time
}

type Writer struct {
	delay     time.Duration
	queue     chan *packet
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        sync.WaitGroup
}

func NewWriter(delay time.Duration) netsim.Node {
	ctx, cancel := context.WithCancel(context.Background())
	dw := &Writer{
		delay:     delay,
		queue:     make(chan *packet, 100),
		wg:        sync.WaitGroup{},
		ctx:       ctx,
		cancelCtx: cancel,
	}
	dw.wg.Go(dw.run)
	return dw
}

// Link implements Node.
func (w *Writer) Link(pw netsim.PacketWriter) netsim.PacketWriter {
	return netsim.PacketWriterFunc(func(b []byte, i netsim.PacketInfo) (int, error) {
		pkt := &packet{
			payload:  make([]byte, len(b)),
			info:     i,
			writer:   pw,
			deadline: time.Now().Add(w.delay),
		}
		n := copy(pkt.payload, b)
		select {
		case w.queue <- pkt:
		default:
			// Deliberately crash here: This is only a delay node, no drop node.
			panic("delay writer overflow: too many writes")
		}
		return n, nil
	})
}

func (w *Writer) run() {
	queue := []*packet{}
	timer := time.NewTimer(0)
	for {
		select {
		case pkt := <-w.queue:
			queue = append(queue, pkt)
		case <-timer.C:
			if len(queue) == 0 {
				continue
			}
			var next *packet
			next, queue = queue[0], queue[1:]
			if _, err := next.writer.WritePacket(next.payload, next.info); err != nil {
				panic(err)
			}
		case <-w.ctx.Done():
			return
		}
		if len(queue) > 0 {
			timer.Reset(time.Until(queue[0].deadline))
		}
	}
}

func (w *Writer) Close() error {
	defer w.wg.Wait()
	w.cancelCtx()
	return nil
}
