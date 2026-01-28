package quic

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/big"
	"net/netip"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/mengelbart/netsim"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"github.com/stretchr/testify/assert"
)

type pathFactory func() []netsim.Node

func pathFactoryFunc(delay time.Duration, bandwidth float64, burst, queueSize int, headDrop bool) pathFactory {
	return func() []netsim.Node {
		nodes := []netsim.Node{}
		if delay > 0 {
			nodes = append(nodes, netsim.NewQueueNode(netsim.NewDelayQueue(delay)))
		}
		if bandwidth > 0 {
			nodes = append(nodes,
				netsim.NewQueueNode(netsim.NewRateQueue(float64(bandwidth), burst, queueSize, headDrop)),
			)
		}
		return nodes
	}
}

func TestQUIC(t *testing.T) {
	bw := float64(1_250_000) // bit/s
	owd := 20 * time.Millisecond
	bdp := int(2 * bw * owd.Seconds())
	cases := []struct {
		name     string
		forward  pathFactory
		backward pathFactory
	}{
		{
			name:     "1",
			forward:  pathFactoryFunc(owd, bw, 5000, bdp, false),
			backward: pathFactoryFunc(owd, bw, 5000, bdp, false),
		},
		{
			name:     "1",
			forward:  pathFactoryFunc(owd, bw, 5000, 10*bdp, false),
			backward: pathFactoryFunc(owd, bw, 5000, 10*bdp, false),
		},
		{
			name:     "1",
			forward:  pathFactoryFunc(owd, bw, 5000, bdp, true),
			backward: pathFactoryFunc(owd, bw, 5000, bdp, true),
		},
		{
			name:     "1",
			forward:  pathFactoryFunc(owd, bw, 5000, 10*bdp, true),
			backward: pathFactoryFunc(owd, bw, 5000, 10*bdp, true),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				net := netsim.NewNet(tc.forward(), tc.backward())
				// net := netsim.NewNet([]netsim.Node{}, []netsim.Node{})

				left := net.NIC(netsim.LeftLocation, netip.MustParseAddr("10.0.0.1"))
				leftConn, err := left.ListenPacket("udp", "10.0.0.1:8080")
				assert.NoError(t, err)

				right := net.NIC(netsim.RightLocation, netip.MustParseAddr("10.0.0.2"))
				rightConn, err := right.Dial("udp", "10.0.0.1:8080")
				assert.NoError(t, err)

				serverTransport := quic.Transport{Conn: leftConn}
				defer serverTransport.Close()
				clientTransport := quic.Transport{Conn: rightConn}
				defer clientTransport.Close()

				var wg sync.WaitGroup
				wg.Go(func() {
					listener, err := serverTransport.Listen(
						generateTLSConfig(),
						&quic.Config{
							Tracer: qlog.DefaultConnectionTracer,
						},
					)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					defer listener.Close()
					conn, err := listener.Accept(ctx)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					defer conn.CloseWithError(0, "by")
					stream, err := conn.AcceptStream(ctx)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					defer stream.Close()
					_, err = io.Copy(io.Discard, stream)
					assert.ErrorIs(t, err, &quic.StreamError{
						StreamID:  stream.StreamID(),
						ErrorCode: 0,
						Remote:    true,
					})
				})

				conn, err := clientTransport.Dial(
					ctx,
					rightConn.RemoteAddr(),
					&tls.Config{InsecureSkipVerify: true, NextProtos: []string{"simtest"}},
					&quic.Config{
						Tracer: qlog.DefaultConnectionTracer,
					},
				)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				defer conn.CloseWithError(0, "bye")
				stream, err := conn.OpenStream()
				assert.NoError(t, err)
				if err != nil {
					return
				}
				defer stream.Close()

				buf := make([]byte, 1_500_000)
				end := time.Now().Add(10 * time.Second)
				for time.Now().Before(end) {
					_, err := stream.Write(buf)
					assert.NoError(t, err)
					if err != nil {
						break
					}
				}
				stream.CancelWrite(0)

				cancel()
				wg.Wait()
				net.Close()
				synctest.Wait()
			})
		})
	}
}

func generateTLSConfig() *tls.Config {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  priv,
		}},
		NextProtos: []string{"simtest"},
	}
}
