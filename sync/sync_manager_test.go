// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/harness"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
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

func generateMockNetwork(numNodes int) (*mockNetwork, error) {
	mn := mocknet.New()

	testHarness, err := harness.NewTestHarness(harness.DefaultOptions())
	if err != nil {
		return nil, err
	}
	err = testHarness.GenerateBlocks(25000)
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
	net, err := generateMockNetwork(20)
	assert.NoError(t, err)

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(net.harness.Blockchain().Params()))
	assert.NoError(t, err)

	node, err := makeMockNode(net.mn, chain)
	assert.NoError(t, err)

	manager := NewSyncManager(context.Background(), chain, node.network, chain.Params(), node.service, nil)

	assert.NoError(t, net.mn.LinkAll())
	assert.NoError(t, net.mn.ConnectAllButSelf())

	manager.Start()

	block, height, _ := chain.BestBlock()
	block2, height2, _ := net.harness.Blockchain().BestBlock()
	assert.Equal(t, block2, block)
	assert.Equal(t, height2, height)
}
