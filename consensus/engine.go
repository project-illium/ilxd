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
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/wire"
	"google.golang.org/protobuf/proto"
	"io"
	"math/rand"
	"sync"
	"time"
)

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
	AvalancheMaxInflightPoll = AvalancheFinalizationScore

	// DeleteInventoryAfter is the maximum time we'll keep a block in memory
	// if it hasn't been finalized by avalanche.
	DeleteInventoryAfter = time.Hour * 6

	// ConsensusProtocol is the libp2p network protocol ID
	ConsensusProtocol = "/consensus/"

	// ConsensusProtocolVersion is the version of the ConsensusProtocol
	ConsensusProtocolVersion = "1.0.0"

	// MinConnectedStakeThreshold is the minimum percentage of the weighted stake
	// set we must be connected to in order to finalize blocks.
	MinConnectedStakeThreshold = .5
)

// requestExpirationMsg signifies a request has expired and
// should be removed from the map.
type requestExpirationMsg struct {
	key string
	p   peer.ID
}

// queryMsg signifies a query from another peer.
type queryMsg struct {
	request    *wire.MsgAvaRequest
	respChan   chan *wire.MsgAvaResponse
	remotePeer peer.ID
}

// newBlockMessage represents new work for the engine.
type newBlockMessage struct {
	header       *blocks.BlockHeader
	isAcceptable bool
	callback     chan<- Status
}

// registerVotesMsg signifies a response to a query from another peer.
type registerVotesMsg struct {
	p    peer.ID
	resp *wire.MsgAvaResponse
}

// RequestBlockFunc is called when the engine receives a query from a peer about
// and unknown block. It should attempt to download the block from the remote peer,
// validate it, then pass it into the engine.
type RequestBlockFunc func(blockID types.ID, remotePeer peer.ID)

// GetBlockIDFunc returns the blockID at the given height or an error if it's not found.
type GetBlockIDFunc func(height uint32) (types.ID, error)

// ConsensusEngine implements a form of the avalanche consensus protocol.
// It primarily consists of an event loop that polls the weighted list of
// validators for any unfinalized blocks and records the responses. Blocks
// finalize when the confidence level exceeds the threshold.
type ConsensusEngine struct {
	ctx          context.Context
	network      *net.Network
	params       *params.NetworkParams
	chooser      *BackoffChooser
	ms           net.MessageSender
	valConn      ValidatorSetConnection
	self         peer.ID
	wg           sync.WaitGroup
	requestBlock RequestBlockFunc
	getBlockID   GetBlockIDFunc
	quit         chan struct{}
	msgChan      chan interface{}
	print        bool

	blocks    map[uint32]*BlockChoice
	queries   map[string]RequestRecord
	callbacks map[types.ID]chan<- Status
}

// NewConsensusEngine returns a new ConsensusEngine
func NewConsensusEngine(ctx context.Context, opts ...Option) (*ConsensusEngine, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	eng := &ConsensusEngine{
		ctx:          ctx,
		network:      cfg.network,
		valConn:      cfg.valConn,
		chooser:      NewBackoffChooser(cfg.chooser),
		params:       cfg.params,
		self:         cfg.self,
		ms:           net.NewMessageSender(cfg.network.Host(), cfg.params.ProtocolPrefix+ConsensusProtocol+ConsensusProtocolVersion),
		wg:           sync.WaitGroup{},
		requestBlock: cfg.requestBlock,
		getBlockID:   cfg.getBlockIDFunc,
		quit:         make(chan struct{}),
		msgChan:      make(chan interface{}),
		blocks:       make(map[uint32]*BlockChoice),
		queries:      make(map[string]RequestRecord),
		callbacks:    make(map[types.ID]chan<- Status),
	}
	eng.network.Host().SetStreamHandler(eng.params.ProtocolPrefix+ConsensusProtocol+ConsensusProtocolVersion, eng.HandleNewStream)
	eng.wg.Add(1)
	go eng.handler()
	return eng, nil
}

// Close gracefully shutsdown the consensus engine
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
				eng.handleRequestExpiration(msg.key, msg.p)
			case *queryMsg:
				eng.handleQuery(msg.request, msg.remotePeer, msg.respChan)
			case *newBlockMessage:
				eng.handleNewBlock(msg.header, msg.isAcceptable, msg.callback)
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

// NewBlock is used to pass new work in the engine. The callback channel will return the final
// status (either Finalized or Rejected). Unfinalized but NotPreffered blocks will remain active
// in the engine until a conflicting block at the same height is finalized. At that point the block
// will be marked as Rejected.
func (eng *ConsensusEngine) NewBlock(header *blocks.BlockHeader, isAcceptable bool, callback chan<- Status) {
	headerCpy := proto.Clone(header).(*blocks.BlockHeader)
	eng.msgChan <- &newBlockMessage{
		header:       headerCpy,
		isAcceptable: isAcceptable,
		callback:     callback,
	}
}

func (eng *ConsensusEngine) handleNewBlock(header *blocks.BlockHeader, isAcceptable bool, callback chan<- Status) {
	blockID := header.ID().Clone()

	bc, ok := eng.blocks[header.Height]
	if !ok {
		bc = NewBlockChoice(header.Height)
		eng.blocks[header.Height] = bc
	}

	if bc.HasBlock(blockID) {
		return
	}

	bc.AddNewBlock(blockID, isAcceptable)

	if len(bc.blockVotes) > 0 {
		log.Debugf("[CONSENSUS] Conflicting blocks at height %d: conflicts %d, block %s", header.Height, len(bc.blockVotes), header.ID())
	}

	eng.callbacks[blockID] = callback
}

// HandleNewStream handles incoming streams from peers. We use one stream for
// incoming and a separate one for outgoing.
func (eng *ConsensusEngine) HandleNewStream(s inet.Stream) {
	go eng.handleNewMessage(s)
}

func (eng *ConsensusEngine) handleNewMessage(s inet.Stream) {
	defer s.Close()
	contextReader := ctxio.NewReader(eng.ctx, s)
	reader := msgio.NewVarintReaderSize(contextReader, inet.MessageSizeMax)
	remotePeer := s.Conn().RemotePeer()
	defer reader.Close()
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-eng.ctx.Done():
			return
		case <-ticker.C:
			return
		default:
		}

		req := new(wire.MsgAvaRequest)
		msgBytes, err := reader.ReadMsg()
		if err != nil {
			reader.ReleaseMsg(msgBytes)
			if err == io.EOF || err == inet.ErrReset {
				s.Close()
				return
			}
			log.Debugf("Error reading from avalanche stream: peer: %s, error: %s", remotePeer, err.Error())
			s.Reset()
			return
		}
		if err := proto.Unmarshal(msgBytes, req); err != nil {
			reader.ReleaseMsg(msgBytes)
			log.Debugf("Error unmarshalling avalanche message: peer: %s, error: %s", remotePeer, err.Error())
			s.Reset()
			return
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
		ticker.Reset(time.Minute)
	}
}

func (eng *ConsensusEngine) handleQuery(req *wire.MsgAvaRequest, remotePeer peer.ID, respChan chan *wire.MsgAvaResponse) {
	if len(req.Heights) == 0 {
		log.Debugf("Received empty avalanche request from peer %s", remotePeer)
		eng.network.IncreaseBanscore(remotePeer, 30, 0)
		return
	}
	resp := &wire.MsgAvaResponse{
		Request_ID: req.Request_ID,
		Votes:      make([][]byte, 0, len(req.Heights)),
	}

	for _, height := range req.Heights {
		preference := types.ID{}
		record, ok := eng.blocks[height]
		if !ok {
			blockID, err := eng.getBlockID(height)
			if err == nil {
				preference = blockID
			}
		} else {
			preference = record.GetPreference()
		}

		resp.Votes = append(resp.Votes, preference.Bytes())
	}

	respChan <- resp
}

func (eng *ConsensusEngine) handleRequestExpiration(key string, p peer.ID) {
	eng.chooser.RegisterDialFailure(p)
	r, ok := eng.queries[key]
	if !ok {
		return
	}
	delete(eng.queries, key)
	heights := r.GetHeights()
	for _, height := range heights {
		bc, ok := eng.blocks[height]
		if ok {
			bc.inflightRequests--
		}
	}
}

func (eng *ConsensusEngine) queueMessageToPeer(req *wire.MsgAvaRequest, peer peer.ID) {
	var (
		key  = queryKey(req.Request_ID, peer.String())
		resp = new(wire.MsgAvaResponse)
	)

	if peer != eng.self {
		err := eng.ms.SendRequest(eng.ctx, peer, req, resp)
		if err != nil {
			eng.msgChan <- &requestExpirationMsg{key, peer}
			return
		}
	} else {
		// Sleep here to not artificially advantage our own node.
		time.Sleep(time.Millisecond * 20)

		respCh := make(chan *wire.MsgAvaResponse)
		eng.msgChan <- &queryMsg{
			request:    req,
			remotePeer: peer,
			respChan:   respCh,
		}
		resp = <-respCh
	}

	eng.msgChan <- &registerVotesMsg{
		p:    peer,
		resp: resp,
	}
}

func (eng *ConsensusEngine) handleRegisterVotes(p peer.ID, resp *wire.MsgAvaResponse) {
	eng.chooser.RegisterDialSuccess(p)
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

	heights := r.GetHeights()
	if len(resp.Votes) != len(heights) {
		log.Debugf("Received avalanche response from peer %s with incorrect number of height votes", p)
		eng.network.IncreaseBanscore(p, 30, 0)
		return
	}

	for i, height := range heights {
		bc, ok := eng.blocks[height]
		if !ok {
			// We are not voting on this anymore
			continue
		}
		bc.inflightRequests--
		if bc.HasFinalized() {
			continue
		}

		if len(resp.Votes[i]) != hash.HashSize {
			log.Debugf("Received avalanche response from peer %s with incorrect hash len", p)
			eng.network.IncreaseBanscore(p, 30, 0)
			continue
		}

		voteID := types.NewID(resp.Votes[i])

		_, ok = bc.blockVotes[voteID]
		if !ok {
			// If we don't know about this block let's request
			// it and also record it as an unknown vote.
			go eng.requestBlock(voteID, p)
			voteID = types.ID{}
		}

		// Block finalized, fire callbacks
		if finalizedID, ok := bc.RecordVote(voteID); ok {
			callback, ok := eng.callbacks[finalizedID]
			if ok && callback != nil {
				delete(eng.callbacks, finalizedID)
				go func(cb chan<- Status) {
					cb <- StatusFinalized
				}(callback)
			}

			for id := range bc.blockVotes {
				if id.Compare(finalizedID) != 0 {
					callback, ok := eng.callbacks[id]
					if ok && callback != nil {
						delete(eng.callbacks, id)
						go func(cb chan<- Status) {
							callback <- StatusRejected
						}(callback)
					}
				}
			}
		}
	}
}

func (eng *ConsensusEngine) pollLoop() {
	if eng.valConn.ConnectedStakePercentage() < MinConnectedStakeThreshold {
		return
	}
	p := eng.chooser.WeightedRandomValidator()
	if p == "" {
		return
	}

	var heights []uint32
	for height, record := range eng.blocks {
		if time.Since(record.timestamp) > DeleteInventoryAfter {
			delete(eng.blocks, height)
			continue
		}

		if record.HasFinalized() {
			continue
		}

		if record.inflightRequests+1 > record.VotesNeededToFinalize() {
			continue
		}

		record.inflightRequests++
		heights = append(heights, height)
	}
	if len(heights) == 0 {
		return
	}

	requestID := rand.Uint32()

	key := queryKey(requestID, p.String())
	eng.queries[key] = NewRequestRecord(time.Now().Unix(), heights)

	req := &wire.MsgAvaRequest{
		Request_ID: requestID,
		Heights:    heights,
	}

	go eng.queueMessageToPeer(req, p)
}

func queryKey(requestID uint32, peerID string) string {
	return fmt.Sprintf("%d|%s", requestID, peerID)
}
