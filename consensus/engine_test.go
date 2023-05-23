// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"math"
	rand "math/rand"
	"testing"
	"time"
)

// MockChooser is a mock WeightedChooser for testing.
type MockChooser struct {
	network *net.Network
}

// WeightedRandomValidator returns a validator weighted by their current stake.
func (m *MockChooser) WeightedRandomValidator() peer.ID {
	peers := m.network.Host().Network().Peers()
	l := len(peers)
	if l == 0 {
		return ""
	}
	i := rand.Intn(l)
	return peers[i]
}

type mockNode struct {
	engine *ConsensusEngine
}

func newMockNode(mn mocknet.Mocknet) (*mockNode, error) {
	host, err := mn.GenPeer()
	if err != nil {
		return nil, err
	}
	network, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(mock.NewMapDatastore()),
	}...)
	if err != nil {
		return nil, err
	}
	engine, err := NewConsensusEngine(context.Background(),
		Params(&params.RegestParams),
		Network(network),
		Chooser(&MockChooser{network: network}),
		HasBlock(func(id types.ID) bool { return false }),
		RequestBlock(func(id types.ID, id2 peer.ID) {}),
	)
	if err != nil {
		return nil, err
	}
	return &mockNode{engine: engine}, nil
}

func TestConsensusEngine(t *testing.T) {
	rand.Seed(time.Now().Unix())
	mn := mocknet.New()
	numNodes := 100
	nodes := make([]*mockNode, 0, numNodes)

	for i := 0; i < numNodes; i++ {
		node, err := newMockNode(mn)
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	testNode, err := newMockNode(mn)
	assert.NoError(t, err)

	assert.NoError(t, mn.LinkAll())
	assert.NoError(t, mn.ConnectAllButSelf())

	t.Run("Test block finalization when all nodes agree", func(t *testing.T) {
		blk1 := &blocks.Block{Header: &blocks.BlockHeader{Height: 1}}
		for _, node := range nodes {
			vr := NewVoteRecord(blk1.ID(), blk1.Header.Height, true)
			vr.confidence = math.MaxUint16
			node.engine.voteRecords[blk1.ID()] = vr
		}

		cb := make(chan Status)
		testNode.engine.NewBlock(blk1.Header, true, cb)
		select {
		case status := <-cb:
			assert.Equal(t, status, StatusFinalized)
		case <-time.After(time.Second * 10):
			t.Errorf("Failed to finalized block 1")
		}
	})

	t.Run(" Test block finalization when all nodes reject", func(t *testing.T) {
		blk2 := &blocks.Block{Header: &blocks.BlockHeader{Height: 2}}
		for _, node := range nodes {
			vr := NewVoteRecord(blk2.ID(), blk2.Header.Height, false)
			vr.confidence = 65534
			node.engine.voteRecords[blk2.ID()] = vr
		}

		cb := make(chan Status)
		testNode.engine.NewBlock(blk2.Header, true, cb)
		select {
		case <-cb:
			t.Errorf("Callback should not have been called block 2")
		case <-time.After(time.Second * 5):
			assert.False(t, testNode.engine.voteRecords[blk2.ID()].isPreferred())
		}
	})

	t.Run("Test block finalization of all nodes with initial preference yes", func(t *testing.T) {
		cb := make(chan Status)
		blk3 := &blocks.Block{Header: &blocks.BlockHeader{Height: 3}}
		for _, node := range nodes {
			node.engine.NewBlock(blk3.Header, true, cb)
		}

		testNode.engine.NewBlock(blk3.Header, true, cb)

		count := 0
		ticker := time.NewTicker(time.Second * 10)
	loop:
		for {
			select {
			case status := <-cb:
				assert.Equal(t, status, StatusFinalized)
				count++
				if count == 101 {
					break loop
				}
			case <-ticker.C:
				t.Errorf("Failed to finalize block 3 for all nodes")
			}
		}
	})

	t.Run("Test block finalization of all nodes with initial preference no", func(t *testing.T) {
		cb := make(chan Status)
		blk4 := &blocks.Block{Header: &blocks.BlockHeader{Height: 4}}
		for _, node := range nodes {
			node.engine.NewBlock(blk4.Header, false, cb)
		}

		testNode.engine.NewBlock(blk4.Header, false, cb)

		ticker := time.NewTicker(time.Second * 5)
	loop:
		for {
			select {
			case <-cb:
				t.Errorf("Callback should not have been called for block 4")
			case <-ticker.C:
				assert.False(t, testNode.engine.voteRecords[blk4.ID()].isPreferred())

				for _, n := range nodes {
					assert.False(t, n.engine.voteRecords[blk4.ID()].isPreferred())
				}
				break loop
			}
		}
	})

	t.Run("Test block finalization of all nodes with random initial preference", func(t *testing.T) {
		cb := make(chan Status)
		blk5 := &blocks.Block{Header: &blocks.BlockHeader{Height: 5}}
		for _, node := range nodes {
			node.engine.NewBlock(blk5.Header, rand.Intn(2) == 1, cb)
		}

		cb2 := make(chan Status)
		testNode.engine.NewBlock(blk5.Header, rand.Intn(2) == 1, cb2)

		var yes bool
		ticker := time.NewTicker(time.Second * 5)
		select {
		case status := <-cb2:
			assert.Equal(t, status, StatusFinalized)
			yes = true
		case <-ticker.C:
			assert.False(t, testNode.engine.voteRecords[blk5.ID()].isPreferred())
			yes = false
		}

		count := 0
	loop:
		for {
			select {
			case status := <-cb:
				if yes {
					assert.Equal(t, status, StatusFinalized)
					count++
					if count == 100 {
						break loop
					}
				} else {
					t.Errorf("Callback should not have been called for block 5")
				}
			case <-ticker.C:
				if !yes {
					for _, n := range nodes {
						assert.False(t, n.engine.voteRecords[blk5.ID()].isPreferred())
					}
					break loop
				} else {
					t.Errorf("Failed to finalize block 5 for all nodes")
				}
			}
		}
	})
	t.Run("Test block finalization of conflicting blocks", func(t *testing.T) {
		blk6a := &blocks.Block{Header: &blocks.BlockHeader{Version: 0, Height: 6}}
		blk6b := &blocks.Block{Header: &blocks.BlockHeader{Version: 1, Height: 6}}
		cb := make(chan Status)
		for _, node := range nodes {
			node.engine.NewBlock(blk6a.Header, true, cb)
		}

		<-time.After(time.Second * 5)

		cba2 := make(chan Status)
		cbb2 := make(chan Status)
		preferred := rand.Intn(2) == 1
		testNode.engine.NewBlock(blk6a.Header, preferred, cba2)
		testNode.engine.NewBlock(blk6b.Header, !preferred, cbb2)

		ticker := time.NewTicker(time.Second * 1000)
		select {
		case status := <-cba2:
			assert.Equal(t, status, StatusFinalized)
		case <-ticker.C:
			t.Errorf("Failed to finalize block 6a for test node")
		}
		select {
		case status := <-cbb2:
			assert.Equal(t, status, StatusRejected)
			assert.False(t, testNode.engine.voteRecords[blk6b.ID()].isPreferred())
		case <-ticker.C:
			t.Errorf("Failed to reject block 6b for test node")
		}
	})
}
