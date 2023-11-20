// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
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

type MockValConn struct{}

func (m *MockValConn) ConnectedStakePercentage() float64 {
	return 100
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
		net.MaxMessageSize(repo.DefaultMaxMessageSize),
	}...)
	if err != nil {
		return nil, err
	}
	engine, err := NewConsensusEngine(context.Background(),
		Params(&params.RegestParams),
		Network(network),
		ValidatorConnector(&MockValConn{}),
		Chooser(&MockChooser{network: network}),
		GetBlockID(func(height uint32) (types.ID, error) { return types.ID{}, errors.New("not found") }),
		RequestBlock(func(id types.ID, id2 peer.ID) {}),
		PeerID(network.Host().ID()),
	)
	if err != nil {
		return nil, err
	}
	return &mockNode{engine: engine}, nil
}

func setup() ([]*mockNode, *mockNode, func(), error) {
	mn := mocknet.New()
	numNodes := 100
	nodes := make([]*mockNode, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		node, err := newMockNode(mn)
		if err != nil {
			return nil, nil, nil, err
		}
		nodes = append(nodes, node)
	}

	testNode, err := newMockNode(mn)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := mn.LinkAll(); err != nil {
		return nil, nil, nil, err
	}
	if err := mn.ConnectAllButSelf(); err != nil {
		return nil, nil, nil, err
	}
	teardown := func() {
		for _, n := range nodes {
			n.engine.Close()
		}
		testNode.engine.Close()
		mn.Close()
	}
	return nodes, testNode, teardown, nil
}

func TestConsensusEngine(t *testing.T) {
	t.Run("Test block finalization when all nodes agree", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		blk1 := &blocks.Block{Header: &blocks.BlockHeader{Height: 1}}
		for _, node := range nodes {
			node.engine.NewBlock(blk1.Header, true, nil)
		}

		cb := make(chan Status)
		testNode.engine.NewBlock(blk1.Header, true, cb)
		select {
		case status := <-cb:
			assert.Equal(t, status, StatusFinalized)
		case <-time.After(time.Second * 30):
			t.Errorf("Failed to finalized block 1")
		}
	})

	t.Run(" Test block finalization when all nodes reject", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		blk2 := &blocks.Block{Header: &blocks.BlockHeader{Height: 2}}
		for _, node := range nodes {
			node.engine.NewBlock(blk2.Header, false, nil)
		}

		cb := make(chan Status)
		testNode.engine.NewBlock(blk2.Header, false, cb)
		select {
		case <-cb:
			t.Fatal("Callback should not have been called block 2")
		case <-time.After(time.Second * 5):
			assert.Equal(t, StatusNotPreferred, testNode.engine.blocks[blk2.Header.Height].blockVotes[blk2.ID()].Status())
		}
	})
	t.Run("Test block finalization of all nodes with initial preference yes", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		cb := make(chan Status)
		blk3 := &blocks.Block{Header: &blocks.BlockHeader{Height: 3}}
		for _, node := range nodes {
			node.engine.NewBlock(blk3.Header, true, cb)
		}

		testNode.engine.NewBlock(blk3.Header, true, cb)

		count := 0
		ticker := time.NewTicker(time.Second * 30)
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
				t.Fatal("Failed to finalize block 3 for all nodes")
			}
		}
	})

	t.Run("Test block finalization of all nodes with initial preference no", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		cb := make(chan Status)
		blk4 := &blocks.Block{Header: &blocks.BlockHeader{Height: 4}}
		for _, node := range nodes {
			node.engine.NewBlock(blk4.Header, false, cb)
		}

		testNode.engine.NewBlock(blk4.Header, false, cb)

		ticker := time.NewTicker(time.Second * 5)
		select {
		case <-cb:
			t.Fatal("Callback should not have been called for block 4")
		case <-ticker.C:
			assert.Equal(t, StatusNotPreferred, testNode.engine.blocks[blk4.Header.Height].blockVotes[blk4.ID()].Status())
			for _, n := range nodes {
				assert.Equal(t, StatusNotPreferred, n.engine.blocks[blk4.Header.Height].blockVotes[blk4.ID()].Status())
			}
		}
	})

	t.Run("Test block finalization of all nodes with random initial preference", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		cb := make(chan Status)
		blk5 := &blocks.Block{Header: &blocks.BlockHeader{Height: 5}}
		for _, node := range nodes {
			node.engine.NewBlock(blk5.Header, rand.Intn(2) == 1, cb)
		}

		cb2 := make(chan Status)
		testNode.engine.NewBlock(blk5.Header, rand.Intn(2) == 1, cb2)

		var yes bool
		ticker := time.NewTicker(time.Second * 30)
		select {
		case status := <-cb2:
			assert.Equal(t, status, StatusFinalized)
			yes = true
		case <-ticker.C:
			assert.Equal(t, StatusNotPreferred, testNode.engine.blocks[blk5.Header.Height].blockVotes[blk5.ID()].Status())
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
					t.Fatal("Callback should not have been called for block 5")
				}
			case <-ticker.C:
				if !yes {
					for _, n := range nodes {
						assert.Equal(t, StatusNotPreferred, n.engine.blocks[blk5.Header.Height].blockVotes[blk5.ID()].Status())
					}
					break loop
				} else {
					t.Fatal("Failed to finalize block 5 for all nodes")
				}
			}
		}
	})
	t.Run("Test block finalization of conflicting blocks", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		blk6a := &blocks.Block{Header: &blocks.BlockHeader{Version: 0, Height: 6}}
		blk6b := &blocks.Block{Header: &blocks.BlockHeader{Version: 1, Height: 6}}
		cb := make(chan Status)
		for _, node := range nodes {
			node.engine.NewBlock(blk6a.Header, true, cb)
		}

		cba2 := make(chan Status)
		cbb2 := make(chan Status)
		testNode.engine.NewBlock(blk6b.Header, true, cbb2)
		testNode.engine.NewBlock(blk6a.Header, true, cba2)

		ticker := time.NewTicker(time.Second * 30)
		select {
		case status := <-cba2:
			assert.Equal(t, status, StatusFinalized)
		case <-ticker.C:
			t.Errorf("Failed to finalize block 6a for test node")
		}
		select {
		case status := <-cbb2:
			assert.Equal(t, status, StatusRejected)
			assert.Equal(t, StatusNotPreferred, testNode.engine.blocks[blk6b.Header.Height].blockVotes[blk6b.ID()].Status())
		case <-ticker.C:
			t.Errorf("Failed to reject block 6b for test node")
		}
	})

	t.Run("Test block finalization of all nodes with conflicting blocks", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		blk6a := &blocks.Block{Header: &blocks.BlockHeader{Version: 0, Height: 6}}
		blk6b := &blocks.Block{Header: &blocks.BlockHeader{Version: 1, Height: 6}}
		blks := []*blocks.Block{blk6a, blk6b}
		cb := make(chan Status)
		for _, node := range nodes {
			rand.Shuffle(len(blks), func(i, j int) {
				blks[i], blks[j] = blks[j], blks[i]
			})
			node.engine.NewBlock(blks[0].Header, true, cb)
			node.engine.NewBlock(blks[1].Header, true, cb)
		}

		cb2 := make(chan Status)
		rand.Shuffle(len(blks), func(i, j int) {
			blks[i], blks[j] = blks[j], blks[i]
		})
		testNode.engine.print = true
		testNode.engine.NewBlock(blks[0].Header, true, cb2)
		testNode.engine.NewBlock(blks[1].Header, true, cb2)

		ticker := time.NewTicker(time.Second * 30)
		finalized, rejected := 0, 0
	loop:
		for {
			select {
			case status := <-cb2:
				if status == StatusFinalized {
					finalized++
				} else if status == StatusRejected {
					rejected++
				}
				if finalized == 1 && rejected == 1 {
					break loop
				}
			case <-ticker.C:
				t.Errorf("Failed to finalize or rejecct block 6 for test node")
			}
		}

		finalized, rejected = 0, 0
	loop2:
		for {
			select {
			case status := <-cb:
				if status == StatusFinalized {
					finalized++
				} else if status == StatusRejected {
					rejected++
				}
				if finalized == 100 && rejected == 100 {
					break loop2
				}
			case <-ticker.C:
				t.Errorf("Failed to finalize or reject block 6 for node")
				break loop2
			}
		}
		assert.Equal(t, 100, finalized)
		assert.Equal(t, 100, rejected)

		blockAStatus := testNode.engine.blocks[blk6a.Header.Height].blockVotes[blk6a.ID()].Status()
		blockBStatus := testNode.engine.blocks[blk6b.Header.Height].blockVotes[blk6b.ID()].Status()
		for _, n := range nodes {
			assert.Equal(t, blockAStatus, n.engine.blocks[blk6a.Header.Height].blockVotes[blk6a.ID()].Status())
			assert.Equal(t, blockBStatus, n.engine.blocks[blk6b.Header.Height].blockVotes[blk6b.ID()].Status())
		}
	})
	t.Run("Test block finalization of all nodes with many blocks", func(t *testing.T) {
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		blk6a := &blocks.Block{Header: &blocks.BlockHeader{Version: 0, Height: 6}}
		blk6b := &blocks.Block{Header: &blocks.BlockHeader{Version: 1, Height: 6}}
		blk6c := &blocks.Block{Header: &blocks.BlockHeader{Version: 2, Height: 6}}
		blk6d := &blocks.Block{Header: &blocks.BlockHeader{Version: 3, Height: 6}}
		blk6e := &blocks.Block{Header: &blocks.BlockHeader{Version: 4, Height: 6}}
		blocks := []*blocks.Block{blk6a, blk6b, blk6c, blk6d, blk6e}
		cb := make(chan Status)
		for _, node := range nodes {
			rand.Shuffle(len(blocks), func(i, j int) {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			})
			node.engine.NewBlock(blocks[0].Header, true, cb)
			node.engine.NewBlock(blocks[1].Header, true, cb)
			node.engine.NewBlock(blocks[2].Header, true, cb)
			node.engine.NewBlock(blocks[3].Header, true, cb)
			node.engine.NewBlock(blocks[4].Header, true, cb)
		}

		cb2 := make(chan Status)
		rand.Shuffle(len(blocks), func(i, j int) {
			blocks[i], blocks[j] = blocks[j], blocks[i]
		})
		testNode.engine.print = true
		testNode.engine.NewBlock(blocks[0].Header, true, cb2)
		testNode.engine.NewBlock(blocks[1].Header, true, cb2)
		testNode.engine.NewBlock(blocks[2].Header, true, cb2)
		testNode.engine.NewBlock(blocks[3].Header, true, cb2)
		testNode.engine.NewBlock(blocks[4].Header, true, cb2)

		ticker := time.NewTicker(time.Second * 60)
		finalized, rejected := 0, 0
	loop:
		for {
			select {
			case status := <-cb2:
				if status == StatusFinalized {
					finalized++
				} else if status == StatusRejected {
					rejected++
				}
				if finalized == 1 && rejected == 4 {
					break loop
				}
			case <-ticker.C:
				t.Errorf("Failed to finalize or reject block 6")
				break loop

			}
		}

		blkAStatus := testNode.engine.blocks[blk6a.Header.Height].blockVotes[blk6a.ID()].Status()
		blkBStatus := testNode.engine.blocks[blk6b.Header.Height].blockVotes[blk6b.ID()].Status()
		blkCStatus := testNode.engine.blocks[blk6c.Header.Height].blockVotes[blk6c.ID()].Status()
		blkDStatus := testNode.engine.blocks[blk6d.Header.Height].blockVotes[blk6d.ID()].Status()
		blkEStatus := testNode.engine.blocks[blk6e.Header.Height].blockVotes[blk6e.ID()].Status()

		finalized, rejected = 0, 0
	loop2:
		for {
			select {
			case status := <-cb:
				if status == StatusFinalized {
					finalized++
				} else if status == StatusRejected {
					rejected++
				}
				if finalized == 100 && rejected == 400 {
					break loop2
				}
			case <-ticker.C:
				t.Errorf("Failed to finalize or reject block 6")
				break loop2

			}
		}

		assert.Equal(t, finalized, 100)
		assert.Equal(t, rejected, 400)

		for _, n := range nodes {
			assert.Equal(t, blkAStatus, n.engine.blocks[blk6a.Header.Height].blockVotes[blk6a.ID()].Status())
			assert.Equal(t, blkBStatus, n.engine.blocks[blk6b.Header.Height].blockVotes[blk6b.ID()].Status())
			assert.Equal(t, blkCStatus, n.engine.blocks[blk6c.Header.Height].blockVotes[blk6c.ID()].Status())
			assert.Equal(t, blkDStatus, n.engine.blocks[blk6d.Header.Height].blockVotes[blk6d.ID()].Status())
			assert.Equal(t, blkEStatus, n.engine.blocks[blk6e.Header.Height].blockVotes[blk6e.ID()].Status())
		}
	})
}
