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
	"github.com/project-illium/ilxd/repo"
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
		HasBlock(func(id types.ID) bool { return false }),
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
			vr := NewBlockVoteRecord(blk1.ID(), blk1.Header.Height, true, true)
			vr.confidence = math.MaxUint16
			node.engine.blockRecords[blk1.Header.Height] = &blockRecord{
				height:        blk1.Header.Height,
				timestamp:     time.Now(),
				activeBit:     0,
				preferredBit:  getBit(blk1.ID(), 0),
				finalizedBits: types.ID{},
				bitVotes:      NewBitVoteRecord(blk1.Header.Height),
				blockInventory: map[types.ID]*VoteRecord{
					blk1.ID(): vr,
				},
			}
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
		nodes, testNode, teardown, err := setup()
		assert.NoError(t, err)
		defer teardown()

		blk2 := &blocks.Block{Header: &blocks.BlockHeader{Height: 2}}
		for _, node := range nodes {
			vr := NewBlockVoteRecord(blk2.ID(), blk2.Header.Height, false, false)
			vr.confidence = 65534
			node.engine.blockRecords[blk2.Header.Height] = &blockRecord{
				height:        blk2.Header.Height,
				timestamp:     time.Now(),
				activeBit:     0,
				preferredBit:  0x80,
				finalizedBits: types.ID{},
				bitVotes:      NewBitVoteRecord(blk2.Header.Height),
				blockInventory: map[types.ID]*VoteRecord{
					blk2.ID(): vr,
				},
			}
		}

		cb := make(chan Status)
		testNode.engine.NewBlock(blk2.Header, true, cb)
		select {
		case <-cb:
			t.Fatal("Callback should not have been called block 2")
		case <-time.After(time.Second * 5):
			assert.False(t, testNode.engine.blockRecords[blk2.Header.Height].blockInventory[blk2.ID()].isPreferred())
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
			assert.False(t, testNode.engine.blockRecords[blk4.Header.Height].blockInventory[blk4.ID()].isPreferred())
			for _, n := range nodes {
				assert.False(t, n.engine.blockRecords[blk4.Header.Height].blockInventory[blk4.ID()].isPreferred())
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
		ticker := time.NewTicker(time.Second * 10)
		select {
		case status := <-cb2:
			assert.Equal(t, status, StatusFinalized)
			yes = true
		case <-ticker.C:
			assert.False(t, testNode.engine.blockRecords[blk5.Header.Height].blockInventory[blk5.ID()].isPreferred())
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
						assert.False(t, n.engine.blockRecords[blk5.Header.Height].blockInventory[blk5.ID()].isPreferred())
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
		preferred := rand.Intn(2) == 1
		testNode.engine.NewBlock(blk6a.Header, preferred, cba2)
		testNode.engine.NewBlock(blk6b.Header, !preferred, cbb2)

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
			assert.False(t, testNode.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].isPreferred())
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
		cba := make(chan Status)
		cbb := make(chan Status)
		for _, node := range nodes {
			preferred := rand.Intn(2) == 1
			node.engine.NewBlock(blk6a.Header, preferred, cba)
			node.engine.NewBlock(blk6b.Header, !preferred, cbb)
		}

		cba2 := make(chan Status)
		cbb2 := make(chan Status)
		preferred := rand.Intn(2) == 1
		testNode.engine.NewBlock(blk6a.Header, preferred, cba2)
		testNode.engine.NewBlock(blk6b.Header, !preferred, cbb2)

		var blockAStatus Status
		ticker := time.NewTicker(time.Second * 30)
		select {
		case status := <-cba2:
			blockAStatus = status
		case <-ticker.C:
			t.Errorf("Failed to finalize or rejecct block 6a for test node")
		}
		assert.True(t, blockAStatus == StatusFinalized || blockAStatus == StatusRejected)
		select {
		case status := <-cbb2:
			if blockAStatus == StatusFinalized {
				assert.Equal(t, status, StatusRejected)
				assert.False(t, testNode.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].isPreferred())
			} else {
				assert.Equal(t, status, StatusFinalized)
				assert.True(t, testNode.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].isPreferred())
			}
		case <-ticker.C:
			t.Errorf("Failed to finalize or reject block 6b for test node")
		}

		//testNode.engine.blockRecords[blk6a.Header.Height].blockInventory[blk6a.ID()].printState()
		//testNode.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].printState()

		finalized := 0
		rejected := 0
	loop:
		for {
			select {
			case status := <-cba:
				if blockAStatus == StatusFinalized {
					assert.Equal(t, status, StatusFinalized)
					//assert.True(t, testNode.engine.blockRecords[blk6a.Header.Height].blockInventory[blk6a.ID()].isPreferred())
					finalized++
				} else {
					assert.Equal(t, status, StatusRejected)
					//assert.False(t, testNode.engine.blockRecords[blk6a.Header.Height].blockInventory[blk6a.ID()].isPreferred())
					rejected++
				}
				if finalized == 100 && rejected == 100 {
					break loop
				}
			case status := <-cbb:
				if blockAStatus == StatusFinalized {
					assert.Equal(t, status, StatusRejected)
					//assert.True(t, testNode.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].isPreferred())
					rejected++
				} else {
					assert.Equal(t, status, StatusFinalized)
					//assert.False(t, testNode.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].isPreferred())
					finalized++
				}
				if finalized == 100 && rejected == 100 {
					break loop
				}
			case <-ticker.C:
				t.Errorf("Failed to finalize or rejecct block 6a for test node")
				break loop
			}
		}
		assert.Equal(t, 100, finalized)
		assert.Equal(t, 100, rejected)

		//for _, n := range nodes {
		//n.engine.blockRecords[blk6a.Header.Height].blockInventory[blk6a.ID()].printState()
		//n.engine.blockRecords[blk6b.Header.Height].blockInventory[blk6b.ID()].printState()
		//fmt.Println("**************")
		//}
	})

}
