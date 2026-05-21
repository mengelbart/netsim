package dualq

import (
	"context"
	"sync"
	"time"

	"github.com/mengelbart/netsim"
	"golang.org/x/time/rate"
)

type packet struct {
	payload []byte
	info    netsim.PacketInfo
	writer  netsim.PacketWriter
}

func (p *packet) mark() {
	p.info.ECN = netsim.ECNCE
}

type Node struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	queue     chan *packet
	wg        sync.WaitGroup

	limiter  *rate.Limiter
	linkRate float64
}

func NewRateQueue(bitrate float64, burst int) netsim.Node {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Node{
		ctx:       ctx,
		cancelCtx: cancel,
		queue:     make(chan *packet, 1),
		wg:        sync.WaitGroup{},
		limiter:   rate.NewLimiter(rate.Limit(bitrate/8), burst), // convert bit/s byte/s
		linkRate:  bitrate,
	}
	w.wg.Go(w.run)
	return w
}

func (nd *Node) Link(pw netsim.PacketWriter) netsim.PacketWriter {
	return netsim.PacketWriterFunc(func(b []byte, a netsim.PacketInfo) (int, error) {
		pkt := &packet{
			payload: make([]byte, len(b)),
			info:    a,
			writer:  pw,
		}
		n := copy(pkt.payload, b)
		select {
		case nd.queue <- pkt:
		case <-nd.ctx.Done(): // Abort if node is closed
			return 0, context.Canceled
		}
		return n, nil
	})
}

func (nd *Node) run() {
	queue := newDualPi2(int(nd.linkRate))

	nd.wg.Go(func() {
		queue.RunUpdates(nd.ctx)
	})

	for {
		var timer <-chan time.Time
		if !queue.empty() {
			timer = time.After(time.Until(nd.nextSent()))
		}

		select {
		case <-nd.ctx.Done():
			return
		case pkt := <-nd.queue:
			queue.push(pkt)
		case <-timer:
			pkt := queue.pop()
			if pkt != nil {
				_, _ = pkt.writer.WritePacket(pkt.payload, pkt.info)
				nd.limiter.ReserveN(time.Now(), len(pkt.payload))
			}
		}
	}
}

func (nd *Node) nextSent() time.Time {
	// TODO: currently we check for MTU independent of actual packet size
	mtu := 1200

	now := time.Now()
	if nd.limiter.TokensAt(now) > float64(mtu) {
		return now
	}
	res := nd.limiter.ReserveN(now, mtu)
	delay := res.Delay()
	res.Cancel()
	return now.Add(delay)
}

// Close implements netsim.Node.
func (nd *Node) Close() error {
	nd.cancelCtx()
	nd.wg.Wait()
	return nil
}
