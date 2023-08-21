// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/harness"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type mockNetwork struct {
	nodes   []*mockNode
	mn      mocknet.Mocknet
	harness *harness.TestHarness
}

type mockNode struct {
	chain   *blockchain.Blockchain
	network *net.Network
	service *ChainService
}

func generateMockNetwork(numNodes, numBlocks int) (*mockNetwork, error) {
	mn := mocknet.New()

	testHarness, err := harness.NewTestHarness(harness.DefaultOptions())
	if err != nil {
		return nil, err
	}
	err = testHarness.GenerateBlocks(numBlocks)
	if err != nil {
		return nil, err
	}

	nodes := make([]*mockNode, 0, numNodes)
	for i := 0; i < numNodes; i++ {

		node, err := makeMockNode(mn, testHarness.Blockchain())
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	mocknetwork := &mockNetwork{
		nodes:   nodes,
		mn:      mn,
		harness: testHarness,
	}

	if err := mn.LinkAll(); err != nil {
		return nil, err
	}
	if err := mn.ConnectAllButSelf(); err != nil {
		return nil, err
	}
	return mocknetwork, nil
}

func makeMockNode(mn mocknet.Mocknet, chain *blockchain.Blockchain) (*mockNode, error) {
	ds := mock.NewMapDatastore()

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
		net.Datastore(ds),
		net.BanDuration(time.Hour),
		net.MaxBanscore(100),
		net.MaxMessageSize(repo.DefaultMaxMessageSize),
	}...)
	if err != nil {
		return nil, err
	}

	service := NewChainService(context.Background(), chain.GetBlockByID, chain, network, chain.Params())

	node := &mockNode{
		chain:   chain,
		network: network,
		service: service,
	}
	return node, nil
}

func TestSync(t *testing.T) {
	net, err := generateMockNetwork(20, 25000)
	assert.NoError(t, err)

	t.Run("sync when all nodes agree", func(t *testing.T) {
		chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(net.harness.Blockchain().Params()))
		assert.NoError(t, err)

		b100, err := net.harness.Blockchain().GetBlockByHeight(100)
		assert.NoError(t, err)
		b200, err := net.harness.Blockchain().GetBlockByHeight(200)
		assert.NoError(t, err)
		b300, err := net.harness.Blockchain().GetBlockByHeight(300)
		assert.NoError(t, err)

		chain.Params().Checkpoints = []params.Checkpoint{
			{
				BlockID: b100.ID(),
				Height:  100,
			},
			{
				BlockID: b200.ID(),
				Height:  200,
			},
			{
				BlockID: b300.ID(),
				Height:  300,
			},
		}

		node, err := makeMockNode(net.mn, chain)
		assert.NoError(t, err)

		manager := NewSyncManager(context.Background(), chain, node.network, chain.Params(), node.service, nil, nil)

		assert.NoError(t, net.mn.LinkAll())
		assert.NoError(t, net.mn.ConnectAllButSelf())

		ch := make(chan struct{})
		go func() {
			manager.Start()
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(time.Second * 30):
			t.Fatal("sync timed out")
		}

		block, height, _ := chain.BestBlock()
		block2, height2, _ := net.harness.Blockchain().BestBlock()
		assert.Equal(t, block2, block)
		assert.Equal(t, height2, height)
		node.network.Close()
		chain.Params().Checkpoints = nil
	})

	t.Run("sync with chain fork", func(t *testing.T) {
		harness2, err := net.harness.Clone()
		assert.NoError(t, err)

		err = net.harness.GenerateBlocks(10000)
		assert.NoError(t, err)

		// Chain two will add a second validator that doesn't create any blocks.
		// This should give chain two a worse score than chain one.
		notes := harness2.SpendableNotes()
		commitment := notes[0].Note.Commitment()
		proof, err := harness2.Accumulator().GetProof(commitment[:])
		assert.NoError(t, err)
		nullifier := types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].UnlockingScript.ScriptCommitment, notes[0].UnlockingScript.ScriptParams...)
		root := harness2.Accumulator().Root()
		sk, pk, err := crypto.GenerateEd25519Key(rand.Reader)
		assert.NoError(t, err)
		pid, err := peer.IDFromPublicKey(pk)
		assert.NoError(t, err)
		valBytes, err := pid.Marshal()
		assert.NoError(t, err)
		stakeTx := &transactions.StakeTransaction{
			Validator_ID: valBytes,
			Amount:       uint64(notes[0].Note.Amount),
			Nullifier:    nullifier[:],
			TxoRoot:      root[:],
			Locktime:     0,
			Signature:    nil,
			Proof:        nil,
		}

		sigHash, err := stakeTx.SigHash()
		assert.NoError(t, err)
		sig, err := sk.Sign(sigHash)
		assert.NoError(t, err)
		stakeTx.Signature = sig

		err = harness2.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(stakeTx)}, nil)
		assert.NoError(t, err)
		err = harness2.GenerateBlocks(10000)
		assert.NoError(t, err)

		// Add more nodes following chain 2
		for i := 0; i < 20; i++ {
			_, err := makeMockNode(net.mn, harness2.Blockchain())
			assert.NoError(t, err)
		}

		chain2, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(net.harness.Blockchain().Params()))
		assert.NoError(t, err)

		node2, err := makeMockNode(net.mn, chain2)
		assert.NoError(t, err)

		manager2 := NewSyncManager(context.Background(), chain2, node2.network, chain2.Params(), node2.service, nil, nil)

		assert.NoError(t, net.mn.LinkAll())
		assert.NoError(t, net.mn.ConnectAllButSelf())

		ch := make(chan struct{})
		go func() {
			manager2.Start()
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(time.Second * 30):
			t.Fatal("sync timed out")
		}

		// Node should sync to chain 1
		block, height, _ := chain2.BestBlock()
		block2, height2, _ := net.harness.Blockchain().BestBlock()
		assert.Equal(t, block2, block)
		assert.Equal(t, height2, height)
	})
	net.mn.Close()
}

func TestSyncFromChooser(t *testing.T) {
	net, err := generateMockNetwork(20, 1000)
	assert.NoError(t, err)

	harness2, err := net.harness.Clone()
	assert.NoError(t, err)
	net.harness.GenerateBlocks(50)
	harness2.GenerateBlocks(100)

	choiceID, err := harness2.Blockchain().GetBlockIDByHeight(1001)
	assert.NoError(t, err)

	// Add more nodes following chain 2
	for i := 0; i < 20; i++ {
		_, err := makeMockNode(net.mn, harness2.Blockchain())
		assert.NoError(t, err)
	}

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(net.harness.Blockchain().Params()))
	assert.NoError(t, err)

	node, err := makeMockNode(net.mn, chain)
	assert.NoError(t, err)

	chooser := func(blks []*blocks.Block) (types.ID, error) {
		for _, blk := range blks {
			if blk.ID() == choiceID {
				return blk.ID(), nil
			}
		}
		return types.ID{}, errors.New("choice not found")
	}

	manager := NewSyncManager(context.Background(), chain, node.network, chain.Params(), node.service, chooser, nil)

	assert.NoError(t, net.mn.LinkAll())
	assert.NoError(t, net.mn.ConnectAllButSelf())

	ch := make(chan struct{})
	go func() {
		manager.Start()
		close(ch)
	}()
	select {
	case <-ch:
	case <-time.After(time.Second * 10):
		t.Fatal("sync timed out")
	}

	block, height, _ := chain.BestBlock()
	block2, height2, _ := harness2.Blockchain().BestBlock()
	assert.Equal(t, block2, block)
	assert.Equal(t, height2, height)
	node.network.Close()
}

func TestSyncWithNodesAtDifferentHeights(t *testing.T) {
	net, err := generateMockNetwork(20, 1000)
	assert.NoError(t, err)

	harness2, err := net.harness.Clone()
	assert.NoError(t, err)
	harness2.GenerateBlocks(1)

	choiceID, err := harness2.Blockchain().GetBlockIDByHeight(1001)
	assert.NoError(t, err)

	// Add more nodes following chain 2
	for i := 0; i < 20; i++ {
		_, err := makeMockNode(net.mn, harness2.Blockchain())
		assert.NoError(t, err)
	}

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(net.harness.Blockchain().Params()))
	assert.NoError(t, err)

	node, err := makeMockNode(net.mn, chain)
	assert.NoError(t, err)

	chooser := func(blks []*blocks.Block) (types.ID, error) {
		for _, blk := range blks {
			if blk.ID() == choiceID {
				return blk.ID(), nil
			}
		}
		return types.ID{}, errors.New("choice not found")
	}

	manager := NewSyncManager(context.Background(), chain, node.network, chain.Params(), node.service, chooser, nil)

	assert.NoError(t, net.mn.LinkAll())
	assert.NoError(t, net.mn.ConnectAllButSelf())

	ch := make(chan struct{})
	go func() {
		manager.Start()
		close(ch)
	}()
	select {
	case <-ch:
	case <-time.After(time.Second * 10):
		t.Fatal("sync timed out")
	}

	block, height, _ := chain.BestBlock()
	block2, height2, _ := harness2.Blockchain().BestBlock()
	assert.Equal(t, block2, block)
	assert.Equal(t, height2, height)
	node.network.Close()
}
