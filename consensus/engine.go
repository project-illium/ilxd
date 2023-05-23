// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"context"
	"fmt"
	ctxio "github.com/jbenet/go-context/io"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/wire"
	"google.golang.org/protobuf/proto"
	"io"
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

const (
	// AvalancheRequestTimeout is the amount of time to wait for a response to a
	// query
	AvalancheRequestTimeout = 1 * time.Minute

	// AvalancheFinalizationScore is the confidence score we consider to be final
	AvalancheFinalizationScore = 160

	// AvalancheTimeStep is the amount of time to wait between event ticks
	AvalancheTimeStep = time.Millisecond

	// AvalancheMaxInflightPoll is the max outstanding requests that we can have
	// for any inventory item.
	AvalancheMaxInflightPoll = 10

	// AvalancheMaxElementPoll is the maximum number of invs to send in a single
	// query
	AvalancheMaxElementPoll = 4096

	// DeleteInventoryAfter is the maximum time we'll keep a block in memory
	// if it hasn't been finalized by avalanche.
	DeleteInventoryAfter = time.Hour * 6

	ConsensusProtocol = "consensus"

	MaxRejectedCache = 200
)

// requestExpirationMsg signifies a request has expired and
// should be removed from the map.
type requestExpirationMsg struct {
	key string
}

// queryMsg signifies a query from another peer.
type queryMsg struct {
	request    *wire.MsgAvaRequest
	respChan   chan *wire.MsgAvaResponse
	remotePeer peer.ID
}

type newBlockMessage struct {
	header                      *blocks.BlockHeader
	initialAcceptancePreference bool
	callback                    chan<- Status
}

// registerVotesMsg signifies a response to a query from another peer.
type registerVotesMsg struct {
	p    peer.ID
	resp *wire.MsgAvaResponse
}

type RequestBlockFunc func(blockID types.ID, remotePeer peer.ID)
type HasBlockFunc func(blockID types.ID) bool

type ConsensusEngine struct {
	ctx          context.Context
	network      *net.Network
	params       *params.NetworkParams
	chooser      blockchain.WeightedChooser
	ms           net.MessageSender
	wg           sync.WaitGroup
	requestBlock RequestBlockFunc
	hasBlock     HasBlockFunc
	quit         chan struct{}
	msgChan      chan interface{}

	voteRecords    map[types.ID]*VoteRecord
	conflicts      map[uint32][]types.ID
	rejectedBlocks map[types.ID]struct{}
	queries        map[string]RequestRecord
	callbacks      map[types.ID]chan<- Status
}

func NewConsensusEngine(ctx context.Context, opts ...Option) (*ConsensusEngine, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	eng := &ConsensusEngine{
		ctx:            ctx,
		network:        cfg.network,
		chooser:        cfg.chooser,
		params:         cfg.params,
		ms:             net.NewMessageSender(cfg.network.Host(), cfg.params.ProtocolPrefix+ConsensusProtocol),
		wg:             sync.WaitGroup{},
		requestBlock:   cfg.requestBlock,
		hasBlock:       cfg.hasBlock,
		quit:           make(chan struct{}),
		msgChan:        make(chan interface{}),
		voteRecords:    make(map[types.ID]*VoteRecord),
		rejectedBlocks: make(map[types.ID]struct{}),
		conflicts:      make(map[uint32][]types.ID),
		queries:        make(map[string]RequestRecord),
		callbacks:      make(map[types.ID]chan<- Status),
	}
	eng.network.Host().SetStreamHandler(eng.params.ProtocolPrefix+ConsensusProtocol, eng.HandleNewStream)
	eng.wg.Add(1)
	go eng.handler()
	return eng, nil
}

func (eng *ConsensusEngine) Close() {
	close(eng.quit)
	eng.wg.Wait()
}

func (eng *ConsensusEngine) handler() {
	eventLoopTicker := time.NewTicker(AvalancheTimeStep)
out:
	for {
		select {
		case m := <-eng.msgChan:
			switch msg := m.(type) {
			case *requestExpirationMsg:
				eng.handleRequestExpiration(msg.key)
			case *queryMsg:
				eng.handleQuery(msg.request, msg.remotePeer, msg.respChan)
			case *newBlockMessage:
				eng.handleNewBlock(msg.header, msg.initialAcceptancePreference, msg.callback)
			case *registerVotesMsg:
				eng.handleRegisterVotes(msg.p, msg.resp)
			}
		case <-eventLoopTicker.C:
			eng.pollLoop()
		case <-eng.quit:
			break out
		}
	}
	eventLoopTicker.Stop()
	eng.wg.Done()
}

func (eng *ConsensusEngine) NewBlock(header *blocks.BlockHeader, initialAcceptancePreference bool, callback chan<- Status) {
	eng.msgChan <- &newBlockMessage{
		header:                      header,
		initialAcceptancePreference: initialAcceptancePreference,
		callback:                    callback,
	}
}

func (eng *ConsensusEngine) handleNewBlock(header *blocks.BlockHeader, initialAcceptancePreference bool, callback chan<- Status) {
	blockID := header.ID()
	_, ok := eng.voteRecords[blockID]
	if ok {
		return
	}
	_, ok = eng.rejectedBlocks[blockID]
	if ok {
		return
	}
	_, ok = eng.conflicts[header.Height]
	if !ok {
		eng.conflicts[header.Height] = make([]types.ID, 0, 1)
	}
	// If we already have a preferred block at this height set the initial
	// preference to false.
	for _, conflict := range eng.conflicts[header.Height] {
		if conflict != header.ID() {
			record, ok := eng.voteRecords[conflict]
			if ok {
				if record.isPreferred() {
					initialAcceptancePreference = false
				}
			}
		}
	}
	eng.conflicts[header.Height] = append(eng.conflicts[header.Height], blockID)

	vr := NewVoteRecord(blockID, header.Height, initialAcceptancePreference)
	eng.voteRecords[blockID] = vr

	eng.callbacks[blockID] = callback
}

func (eng *ConsensusEngine) HandleNewStream(s inet.Stream) {
	go eng.handleNewMessage(s)
}

func (eng *ConsensusEngine) handleNewMessage(s inet.Stream) {
	defer s.Close()
	contextReader := ctxio.NewReader(eng.ctx, s)
	reader := msgio.NewVarintReaderSize(contextReader, inet.MessageSizeMax)
	remotePeer := s.Conn().RemotePeer()
	defer reader.Close()

	for {
		select {
		case <-eng.ctx.Done():
			return
		default:
		}

		req := new(wire.MsgAvaRequest)
		msgBytes, err := reader.ReadMsg()
		if err != nil {
			reader.ReleaseMsg(msgBytes)
			s.Reset()
			if err == io.EOF {
				log.Debugf("Peer %s closed avalanche stream", remotePeer)
			}
			return
		}
		if err := proto.Unmarshal(msgBytes, req); err != nil {
			reader.ReleaseMsg(msgBytes)
			continue
		}
		reader.ReleaseMsg(msgBytes)

		respCh := make(chan *wire.MsgAvaResponse)
		eng.msgChan <- &queryMsg{
			request:    req,
			respChan:   respCh,
			remotePeer: remotePeer,
		}

		respMsg := <-respCh
		err = net.WriteMsg(s, respMsg)
		if err != nil {
			log.Errorf("Error writing avalanche stream to peer %d", remotePeer)
			s.Reset()
		}
	}
}

func (eng *ConsensusEngine) handleQuery(req *wire.MsgAvaRequest, remotePeer peer.ID, respChan chan *wire.MsgAvaResponse) {
	votes := make([]byte, len(req.Invs))
	if len(req.Invs) == 0 {
		log.Debugf("Received empty avalanche request from peer %s", remotePeer)
		eng.network.IncreaseBanscore(remotePeer, 30, 0)
		return
	}
	for i, invBytes := range req.Invs {
		inv := types.NewID(invBytes)

		if _, exists := eng.rejectedBlocks[inv]; exists {
			votes[i] = 0x00 // No vote
			continue
		}
		record, ok := eng.voteRecords[inv]
		if ok {
			// We're only going to vote for items we have a record for.
			votes[i] = 0x00 // No vote
			if record.isPreferred() {
				votes[i] = 0x01 // Yes vote
			}
		} else {
			if eng.hasBlock(inv) {
				votes[i] = 0x01
			} else {
				votes[i] = 0x80 // Neutral vote
				// Request to download the block from the remote peer
				go eng.requestBlock(inv, remotePeer)
			}
		}
	}
	resp := &wire.MsgAvaResponse{
		Request_ID: req.Request_ID,
		Votes:      votes,
	}

	respChan <- resp
}

func (eng *ConsensusEngine) handleRequestExpiration(key string) {
	r, ok := eng.queries[key]
	if !ok {
		return
	}
	delete(eng.queries, key)

	invs := r.GetInvs()
	for inv := range invs {
		vr, ok := eng.voteRecords[inv]
		if ok {
			vr.inflightRequests--
		}
	}
}

func (eng *ConsensusEngine) queueMessageToPeer(req *wire.MsgAvaRequest, peer peer.ID) {
	var (
		key  = queryKey(req.Request_ID, peer.String())
		resp = new(wire.MsgAvaResponse)
	)

	err := eng.ms.SendRequest(eng.ctx, peer, req, resp)
	if err != nil {
		log.Errorf("Error reading avalanche response from peer %s", peer.String())
		eng.msgChan <- &requestExpirationMsg{key}
		eng.network.IncreaseBanscore(peer, 0, 10)
		return
	}

	eng.msgChan <- &registerVotesMsg{
		p:    peer,
		resp: resp,
	}
}

func (eng *ConsensusEngine) handleRegisterVotes(p peer.ID, resp *wire.MsgAvaResponse) {
	key := queryKey(resp.Request_ID, p.String())

	r, ok := eng.queries[key]
	if !ok {
		log.Debugf("Received avalanche response from peer %s with an unknown request ID", p)
		eng.network.IncreaseBanscore(p, 30, 0)
		return
	}

	// Always delete the key if it's present
	delete(eng.queries, key)

	if r.IsExpired() {
		log.Debugf("Received avalanche response from peer %s with an expired request", p)
		eng.network.IncreaseBanscore(p, 0, 20)
		return
	}

	invs := r.GetInvs()
	if len(resp.Votes) != len(invs) {
		log.Debugf("Received avalanche response from peer %s with incorrect number of votes", p)
		eng.network.IncreaseBanscore(p, 30, 0)
		return
	}

	i := -1
	for inv := range invs {
		i++
		vr, ok := eng.voteRecords[inv]
		if !ok {
			// We are not voting on this anymore
			continue
		}
		vr.inflightRequests--
		if vr.hasFinalized() {
			continue
		}

		if !vr.regsiterVote(resp.Votes[i]) {
			// This vote did not provide any extra information
			continue
		}

		if vr.isPreferred() {
			// We need to keep track of conflicting blocks
			// when this one becomes accepted we need to set the
			// confidence of the conflicts back to zero.
			for _, conflict := range eng.conflicts[vr.height] {
				if conflict != vr.blockID {
					eng.voteRecords[conflict].Reset(false)
				}
			}
		}

		if vr.status() == StatusFinalized {
			callback, ok := eng.callbacks[inv]
			if ok && callback != nil {
				go func() {
					callback <- vr.status()
				}()
			}
			for _, conflict := range eng.conflicts[vr.height] {
				if conflict != vr.blockID {
					eng.voteRecords[conflict].Reject()
					eng.limitRejected()
					eng.rejectedBlocks[conflict] = struct{}{}
					callback, ok := eng.callbacks[conflict]
					if ok && callback != nil {
						go func() {
							callback <- eng.voteRecords[conflict].status()
						}()
					}
				}
			}
		}
	}
}

func (eng *ConsensusEngine) pollLoop() {
	invs := eng.getInvsForNextPoll()
	if len(invs) == 0 {
		return
	}

	p := eng.chooser.WeightedRandomValidator()
	if p == "" {
		return
	}
	requestID := rand.Uint32()

	key := queryKey(requestID, p.String())
	eng.queries[key] = NewRequestRecord(time.Now().Unix(), invs)

	invList := make([][]byte, 0, len(invs))
	for _, inv := range invs {
		b := make([]byte, len(inv))
		copy(b, inv[:])
		invList = append(invList, b)
	}

	req := &wire.MsgAvaRequest{
		Request_ID: requestID,
		Invs:       invList,
	}

	go eng.queueMessageToPeer(req, p)
}

func (eng *ConsensusEngine) getInvsForNextPoll() []types.ID {
	var (
		invs     []types.ID
		toDelete []types.ID
	)
	for id, r := range eng.voteRecords {
		// Delete very old inventory that hasn't finalized
		if time.Since(r.timestamp) > DeleteInventoryAfter {
			toDelete = append(toDelete, id)
			continue
		}

		if r.hasFinalized() {
			// If this has finalized we can just skip.
			continue
		}
		if r.inflightRequests >= AvalancheMaxInflightPoll {
			// If we are already at the max inflight then continue
			continue
		}
		r.inflightRequests++

		// We don't have a decision, we need more votes.
		invs = append(invs, id)
	}

	if len(invs) >= AvalancheMaxElementPoll {
		invs = invs[:AvalancheMaxElementPoll]
	}

	for _, td := range toDelete {
		delete(eng.voteRecords, td)
	}

	return invs
}

func (eng *ConsensusEngine) limitRejected() {
	if len(eng.rejectedBlocks) >= MaxRejectedCache {
		for blockID := range eng.rejectedBlocks {
			delete(eng.rejectedBlocks, blockID)
			break
		}
	}
}

func queryKey(requestID uint32, peerID string) string {
	return fmt.Sprintf("%d|%s", requestID, peerID)
}
