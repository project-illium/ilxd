// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/google/martian/log"
	ctxio "github.com/jbenet/go-context/io"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio"
	"github.com/project-illium/ilxd/models"
	"github.com/project-illium/ilxd/models/wire"
	"github.com/project-illium/ilxd/net"
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
	AvalancheFinalizationScore = 256

	// AvalancheTimeStep is the amount of time to wait between event ticks
	AvalancheTimeStep = time.Millisecond

	// AvalancheMaxInflightPoll is the max outstanding requests that we can have
	// for any inventory item.
	AvalancheMaxInflightPoll = 10

	// AvalancheMaxElementPoll is the maximum number of invs to send in a single
	// query
	AvalancheMaxElementPoll = 4096

	// DeleteInventoryAfter is the maximum time we'll keep a transaction in memory
	// if it hasn't been finalized by avalanche.
	DeleteInventoryAfter = time.Hour * 6
)

var (
	// AvalancheProtocolID is the protocol ID for exchanging avalanche related messages
	// in the network host. It ensures inbound streams with this ID are routed here.
	AvalancheProtocolID = protocol.ID("/ilxd/avalanche")
)

// requestExpirationMsg signifies a request has expired and
// should be removed from the map.
type requestExpirationMsg struct {
	key string
}

// queryMsg signifies a query from another peer.
type queryMsg struct {
	request  *wire.MsgAvaRequest
	respChan chan *wire.MsgAvaResponse
}

type newBlockMessage struct {
	blockID                     models.ID
	initialAcceptancePreference bool
	callback                    chan<- Status
}

// registerVotesMsg signifies a response to a query from another peer.
type registerVotesMsg struct {
	p    peer.ID
	resp *wire.MsgAvaResponse
}

type AvalancheEngine struct {
	ctx     context.Context
	network *net.Network
	ms      net.MessageSender
	wg      sync.WaitGroup
	quit    chan struct{}
	msgChan chan interface{}

	voteRecords    map[models.ID]*VoteRecord
	rejectedBlocks map[models.ID]struct{}
	queries        map[string]RequestRecord
	callbacks      map[models.ID]chan<- Status
	streams        map[peer.ID]inet.Stream
	start          time.Time

	alwaysNo bool
}

func NewAvalancheEngine(ctx context.Context, network *net.Network) (*AvalancheEngine, error) {
	return &AvalancheEngine{
		ctx:            ctx,
		network:        network,
		ms:             net.NewMessageSender(network.Host(), AvalancheProtocolID),
		wg:             sync.WaitGroup{},
		quit:           make(chan struct{}),
		msgChan:        make(chan interface{}),
		voteRecords:    make(map[models.ID]*VoteRecord),
		rejectedBlocks: make(map[models.ID]struct{}),
		queries:        make(map[string]RequestRecord),
		callbacks:      make(map[models.ID]chan<- Status),
	}, nil
}

// Start begins the core handler which processes peers and avalanche messages.
func (eng *AvalancheEngine) Start() {
	eng.network.Host().SetStreamHandler(AvalancheProtocolID, eng.HandleNewStream)
	eng.wg.Add(1)
	go eng.handler()
}

func (eng *AvalancheEngine) Stop() {
	close(eng.quit)
	eng.wg.Wait()
}

func (eng *AvalancheEngine) handler() {
	eventLoopTicker := time.NewTicker(AvalancheTimeStep)
out:
	for {
		select {
		case m := <-eng.msgChan:
			switch msg := m.(type) {
			case *requestExpirationMsg:
				eng.handleRequestExpiration(msg.key)
			case *queryMsg:
				eng.handleQuery(msg.request, msg.respChan)
			case *newBlockMessage:
				eng.handleNewBlock(msg.blockID, msg.initialAcceptancePreference, msg.callback)
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

func (eng *AvalancheEngine) NewBlock(blockID models.ID, initialAcceptancePreference bool, callback chan<- Status) {
	eng.start = time.Now()
	eng.msgChan <- &newBlockMessage{
		blockID:                     blockID,
		initialAcceptancePreference: initialAcceptancePreference,
		callback:                    callback,
	}
}

func (eng *AvalancheEngine) handleNewBlock(blockID models.ID, initialAcceptancePreference bool, callback chan<- Status) {
	_, ok := eng.voteRecords[blockID]
	if ok {
		return
	}
	_, ok = eng.rejectedBlocks[blockID]
	if ok {
		return
	}

	vr := NewVoteRecord(blockID, initialAcceptancePreference)
	eng.voteRecords[blockID] = vr

	eng.callbacks[blockID] = callback
}

func (eng *AvalancheEngine) HandleNewStream(s inet.Stream) {
	go eng.handleNewMessage(s)
}

func (eng *AvalancheEngine) handleNewMessage(s inet.Stream) {
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
			request:  req,
			respChan: respCh,
		}

		respMsg := <-respCh
		err = net.WriteMsg(s, respMsg)
		if err != nil {
			log.Errorf("Error writing avalanche stream to peer %d", remotePeer)
			s.Reset()
		}
	}
}

func (eng *AvalancheEngine) handleQuery(req *wire.MsgAvaRequest, respChan chan *wire.MsgAvaResponse) {
	votes := make([]byte, len(req.Invs))
	for i, invBytes := range req.Invs {
		inv := models.NewID(invBytes)
		if eng.alwaysNo {
			votes[i] = 0x00
			continue
		}

		if _, exists := eng.rejectedBlocks[inv]; exists {
			votes[i] = 0x00 // No vote
			continue
		}
		record, ok := eng.voteRecords[inv]
		if ok {
			// We're only going to vote for items we have a record for.
			vote := byte(0x00) // No vote
			if record.isPreferred() {
				vote = 0x01 // Yes vote
			}
			votes[i] = vote
		} else {
			// TODO: we need to download this block from the peer and give it to
			// the mempool for processing.

			votes[i] = 0x80 // Neutral vote
		}
	}
	resp := &wire.MsgAvaResponse{
		RequestID: req.RequestID,
		Votes:     votes,
	}

	respChan <- resp
}

func (eng *AvalancheEngine) handleRequestExpiration(key string) {
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

func (eng *AvalancheEngine) queueMessageToPeer(req *wire.MsgAvaRequest, peer peer.ID) {

	var (
		key  = queryKey(req.RequestID, peer.Pretty())
		resp = new(wire.MsgAvaResponse)
	)

	err := eng.ms.SendRequest(eng.ctx, peer, req, resp)
	if err != nil {
		log.Errorf("Error reading avalanche response from peer %s", peer.Pretty())
		eng.msgChan <- &requestExpirationMsg{key}
		return
	}

	eng.msgChan <- &registerVotesMsg{
		p:    peer,
		resp: resp,
	}
}

func (eng *AvalancheEngine) handleRegisterVotes(p peer.ID, resp *wire.MsgAvaResponse) {
	key := queryKey(resp.RequestID, p.Pretty())

	r, ok := eng.queries[key]
	if !ok {
		log.Debugf("Received avalanche response from peer %s with an unknown request ID", p)
		return
	}

	// Always delete the key if it's present
	delete(eng.queries, key)

	if r.IsExpired() {
		log.Debugf("Received avalanche response from peer %s with an expired request", p)
		return
	}

	invs := r.GetInvs()
	if len(resp.Votes) != len(invs) {
		log.Debugf("Received avalanche response from peer %s with incorrect number of votes", p)
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
			// when this one becomes accepted with need to set the
			// confidence of the conflicts back to zero.
		}

		if vr.status() == StatusFinalized || vr.status() == StatusRejected {
			fmt.Println(time.Since(eng.start))
			callback, ok := eng.callbacks[inv]
			if ok {
				callback <- vr.status()
			}
		}
	}
}

func (eng *AvalancheEngine) pollLoop() {
	invs := eng.getInvsForNextPoll()
	if len(invs) == 0 {
		return
	}

	p := eng.getRandomPeerToQuery()
	if p == nil {
		return
	}
	requestID := rand.Uint32()

	key := queryKey(requestID, p.Pretty())
	eng.queries[key] = NewRequestRecord(time.Now().Unix(), invs)

	invList := make([][]byte, 0, len(invs))
	for _, inv := range invs {
		invList = append(invList, inv[:])
	}

	req := &wire.MsgAvaRequest{
		RequestID: requestID,
		Invs:      invList,
	}

	go eng.queueMessageToPeer(req, *p)
}

func (eng *AvalancheEngine) getInvsForNextPoll() []models.ID {
	var invs []models.ID
	var toDelete []models.ID
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

func (eng *AvalancheEngine) getRandomPeerToQuery() *peer.ID {
	peers := eng.network.Host().Network().Peers()
	l := len(peers)
	if l == 0 {
		return nil
	}
	i := rand.Intn(l)
	return &peers[i]
}

func queryKey(requestID uint32, peerID string) string {
	return fmt.Sprintf("%d|%s", requestID, peerID)
}
