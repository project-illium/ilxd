// Copyright (c) 2016 Protocol Labs
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"bufio"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-kad-dht/metrics"
	"github.com/libp2p/go-msgio/pbio"
	"go.opencensus.io/stats"
	"google.golang.org/protobuf/proto"

	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/libp2p/go-msgio"
)

var readMessageTimeout = 10 * time.Second

// ErrReadTimeout is an error that occurs when no message is read within the timeout period.
var ErrReadTimeout = fmt.Errorf("timed out reading response")

// MessageSender handles sending wire protocol messages to a given peer
type MessageSender interface {
	// SendRequest sends a peer a message and waits for its response
	SendRequest(ctx context.Context, p peer.ID, req proto.Message, resp proto.Message) error
	// SendMessage sends a peer a message without waiting on a response
	SendMessage(ctx context.Context, p peer.ID, pmes proto.Message) error
}

// messageSenderImpl is responsible for sending requests and messages to peers efficiently, including reuse of streams.
// It also tracks metrics for sent requests and messages.
type messageSenderImpl struct {
	host      host.Host // the network services we need
	smlk      sync.Mutex
	strmap    map[peer.ID]*peerMessageSender
	protocols []protocol.ID
}

func NewMessageSender(h host.Host, protos ...protocol.ID) MessageSender {
	ms := &messageSenderImpl{
		host:      h,
		strmap:    make(map[peer.ID]*peerMessageSender),
		protocols: protos,
	}

	disconnected := func(_ network.Network, conn network.Conn) {
		ms.OnDisconnect(context.Background(), conn.RemotePeer())
	}

	notifier := &network.NotifyBundle{
		DisconnectedF: disconnected,
	}

	h.Network().Notify(notifier)
	return ms
}

func (m *messageSenderImpl) OnDisconnect(ctx context.Context, p peer.ID) {
	m.smlk.Lock()
	defer m.smlk.Unlock()
	ms, ok := m.strmap[p]
	if !ok {
		return
	}
	delete(m.strmap, p)

	// Do this asynchronously as ms.lk can block for a while.
	go func() {
		if err := ms.lk.Lock(ctx); err != nil {
			return
		}
		defer ms.lk.Unlock()
		ms.invalidate()
	}()
}

// SendRequest sends out a request, but also makes sure to
// measure the RTT for latency measurements.
func (m *messageSenderImpl) SendRequest(ctx context.Context, p peer.ID, req proto.Message, resp proto.Message) error {
	ms, err := m.messageSenderForPeer(ctx, p)
	if err != nil {
		stats.Record(ctx,
			metrics.SentRequests.M(1),
			metrics.SentRequestErrors.M(1),
		)
		log.Debugw("request failed to open message sender", "error", err, "to", p)
		return err
	}

	start := time.Now()

	err = ms.SendRequest(ctx, req, resp)
	if err != nil {
		stats.Record(ctx,
			metrics.SentRequests.M(1),
			metrics.SentRequestErrors.M(1),
		)
		log.Debugw("request failed", "error", err, "to", p)
		return err
	}

	stats.Record(ctx,
		metrics.SentRequests.M(1),
		metrics.OutboundRequestLatency.M(float64(time.Since(start))/float64(time.Millisecond)),
	)
	m.host.Peerstore().RecordLatency(p, time.Since(start))
	return nil
}

// SendMessage sends out a message
func (m *messageSenderImpl) SendMessage(ctx context.Context, p peer.ID, pmes proto.Message) error {
	ms, err := m.messageSenderForPeer(ctx, p)
	if err != nil {
		stats.Record(ctx,
			metrics.SentMessages.M(1),
			metrics.SentMessageErrors.M(1),
		)
		log.Debugw("message failed to open message sender", "error", err, "to", p)
		return err
	}

	if err := ms.SendMessage(ctx, pmes); err != nil {
		stats.Record(ctx,
			metrics.SentMessages.M(1),
			metrics.SentMessageErrors.M(1),
		)
		log.Debugw("message failed", "error", err, "to", p)
		return err
	}

	stats.Record(ctx,
		metrics.SentMessages.M(1),
	)
	return nil
}

func (m *messageSenderImpl) messageSenderForPeer(ctx context.Context, p peer.ID) (*peerMessageSender, error) {
	m.smlk.Lock()
	ms, ok := m.strmap[p]
	if ok {
		m.smlk.Unlock()
		return ms, nil
	}
	ms = &peerMessageSender{p: p, m: m, lk: NewCtxMutex()}
	m.strmap[p] = ms
	m.smlk.Unlock()

	if err := ms.prepOrInvalidate(ctx); err != nil {
		m.smlk.Lock()
		defer m.smlk.Unlock()

		if msCur, ok := m.strmap[p]; ok {
			// Changed. Use the new one, old one is invalid and
			// not in the map so we can just throw it away.
			if ms != msCur {
				return msCur, nil
			}
			// Not changed, remove the now invalid stream from the
			// map.
			delete(m.strmap, p)
		}
		// Invalid but not in map. Must have been removed by a disconnect.
		return nil, err
	}
	// All ready to go.
	return ms, nil
}

// peerMessageSender is responsible for sending requests and messages to a particular peer
type peerMessageSender struct {
	s  network.Stream
	r  msgio.ReadCloser
	lk CtxMutex
	p  peer.ID
	m  *messageSenderImpl

	invalid   bool
	singleMes int
}

// invalidate is called before this peerMessageSender is removed from the strmap.
// It prevents the peerMessageSender from being reused/reinitialized and then
// forgotten (leaving the stream open).
func (ms *peerMessageSender) invalidate() {
	ms.invalid = true
	if ms.s != nil {
		_ = ms.s.Reset()
		ms.s = nil
	}
}

func (ms *peerMessageSender) prepOrInvalidate(ctx context.Context) error {
	if err := ms.lk.Lock(ctx); err != nil {
		return err
	}
	defer ms.lk.Unlock()

	if err := ms.prep(ctx); err != nil {
		ms.invalidate()
		return err
	}
	return nil
}

func (ms *peerMessageSender) prep(ctx context.Context) error {
	if ms.invalid {
		return fmt.Errorf("message sender has been invalidated")
	}
	if ms.s != nil {
		return nil
	}

	// We only want to speak to peers using our primary protocols. We do not want to query any peer that only speaks
	// one of the secondary "server" protocols that we happen to support (e.g. older nodes that we can respond to for
	// backwards compatibility reasons).
	nstr, err := ms.m.host.NewStream(ctx, ms.p, ms.m.protocols...)
	if err != nil {
		return err
	}

	ms.r = msgio.NewVarintReaderSize(nstr, network.MessageSizeMax)
	ms.s = nstr

	return nil
}

// streamReuseTries is the number of times we will try to reuse a stream to a
// given peer before giving up and reverting to the old one-message-per-stream
// behaviour.
const streamReuseTries = 3

func (ms *peerMessageSender) SendMessage(ctx context.Context, pmes proto.Message) error {
	if err := ms.lk.Lock(ctx); err != nil {
		return err
	}
	defer ms.lk.Unlock()

	retry := false
	for {
		if err := ms.prep(ctx); err != nil {
			return err
		}

		if err := ms.writeMsg(pmes); err != nil {
			_ = ms.s.Reset()
			ms.s = nil

			if retry {
				log.Debugw("error writing message", "error", err)
				return err
			}
			log.Debugw("error writing message", "error", err, "retrying", true)
			retry = true
			continue
		}

		var err error
		if ms.singleMes > streamReuseTries {
			err = ms.s.Close()
			ms.s = nil
		} else if retry {
			ms.singleMes++
		}

		return err
	}
}

func (ms *peerMessageSender) SendRequest(ctx context.Context, req proto.Message, resp proto.Message) error {
	if err := ms.lk.Lock(ctx); err != nil {
		return err
	}
	defer ms.lk.Unlock()

	retry := false
	for {
		if err := ms.prep(ctx); err != nil {
			return err
		}

		if err := ms.writeMsg(req); err != nil {
			_ = ms.s.Reset()
			ms.s = nil

			if retry {
				log.Debugw("error writing message", "error", err)
				return err
			}
			log.Debugw("error writing message", "error", err, "retrying", true)
			retry = true
			continue
		}

		if err := ms.ctxReadMsg(ctx, resp); err != nil {
			_ = ms.s.Reset()
			ms.s = nil

			if retry {
				log.Debugw("error reading message", "error", err)
				return err
			}
			log.Debugw("error reading message", "error", err, "retrying", true)
			retry = true
			continue
		}

		var err error
		if ms.singleMes > streamReuseTries {
			err = ms.s.Close()
			ms.s = nil
		} else if retry {
			ms.singleMes++
		}

		return err
	}
}

func (ms *peerMessageSender) writeMsg(pmes proto.Message) error {
	return WriteMsg(ms.s, pmes)
}

func (ms *peerMessageSender) ctxReadMsg(ctx context.Context, mes proto.Message) error {
	return ReadMsg(ctx, ms.r, mes)
}

// The Protobuf writer performs multiple small writes when writing a message.
// We need to buffer those writes, to make sure that we're not sending a new
// packet for every single write.
type bufferedDelimitedWriter struct {
	*bufio.Writer
	pbio.WriteCloser
}

var writerPool = sync.Pool{
	New: func() interface{} {
		w := bufio.NewWriter(nil)
		return &bufferedDelimitedWriter{
			Writer:      w,
			WriteCloser: pbio.NewDelimitedWriter(w),
		}
	},
}

func WriteMsg(w io.Writer, mes proto.Message) error {
	bw := writerPool.Get().(*bufferedDelimitedWriter)
	bw.Reset(w)
	err := bw.WriteMsg(mes)
	if err == nil {
		err = bw.Flush()
	}
	bw.Reset(nil)
	writerPool.Put(bw)
	return err
}

func (w *bufferedDelimitedWriter) Flush() error {
	return w.Writer.Flush()
}

func ReadMsg(ctx context.Context, r msgio.ReadCloser, mes proto.Message) error {
	errc := make(chan error, 1)
	go func(r msgio.ReadCloser) {
		defer close(errc)
		bytes, err := r.ReadMsg()
		defer r.ReleaseMsg(bytes)
		if err != nil {
			errc <- err
			return
		}
		errc <- proto.Unmarshal(bytes, mes)
	}(r)

	t := time.NewTimer(readMessageTimeout)
	defer t.Stop()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return ErrReadTimeout
	}
}
