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

func TestSyncManagerSyncToCheckpoints(t *testing.T) {
	mn := mocknet.New()

	ds := mock.NewMapDatastore()

	host1, err := mn.GenPeer()
	assert.NoError(t, err)
	network1, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host1),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(ds),
	}...)
	assert.NoError(t, err)

	testHarness1, err := harness.NewTestHarness(harness.DefaultOptions())
	assert.NoError(t, err)

	err = testHarness1.GenerateBlocks(15100)
	assert.NoError(t, err)

	chk1, err := testHarness1.Blockchain().GetBlockByHeight(5000)
	assert.NoError(t, err)
	chk2, err := testHarness1.Blockchain().GetBlockByHeight(10000)
	assert.NoError(t, err)
	chk3, err := testHarness1.Blockchain().GetBlockByHeight(15000)
	assert.NoError(t, err)

	testHarness1.Blockchain().Params().Checkpoints = []params.Checkpoint{
		{BlockID: chk1.ID(), Height: 5000},
		{BlockID: chk2.ID(), Height: 10000},
		{BlockID: chk3.ID(), Height: 15000},
	}

	NewChainService(context.Background(), testHarness1.Blockchain().GetBlockByID, testHarness1.Blockchain(), network1, testHarness1.Blockchain().Params())

	host2, err := mn.GenPeer()
	assert.NoError(t, err)
	network2, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host2),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(ds),
	}...)
	assert.NoError(t, err)

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(testHarness1.Blockchain().Params()))
	assert.NoError(t, err)

	service2 := NewChainService(context.Background(), chain.GetBlockByID, chain, network2, chain.Params())

	assert.NoError(t, mn.LinkAll())
	assert.NoError(t, mn.ConnectAllButSelf())

	sm := NewSyncManager(context.Background(), chain, network2, chain.Params(), service2)

	err = sm.syncToCheckpoints(0)
	assert.NoError(t, err)

	blk, err := chain.GetBlockByHeight(testHarness1.Blockchain().Params().Checkpoints[0].Height)
	assert.NoError(t, err)
	assert.Equal(t, testHarness1.Blockchain().Params().Checkpoints[0].BlockID, blk.ID())

	blk, err = chain.GetBlockByHeight(testHarness1.Blockchain().Params().Checkpoints[1].Height)
	assert.NoError(t, err)
	assert.Equal(t, testHarness1.Blockchain().Params().Checkpoints[1].BlockID, blk.ID())

	blk, err = chain.GetBlockByHeight(testHarness1.Blockchain().Params().Checkpoints[2].Height)
	assert.NoError(t, err)
	assert.Equal(t, testHarness1.Blockchain().Params().Checkpoints[2].BlockID, blk.ID())
}

func TestSyncManagerQueryPeers(t *testing.T) {
	mn := mocknet.New()

	host0, err := mn.GenPeer()
	assert.NoError(t, err)
	network0, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host0),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(mock.NewMapDatastore()),
	}...)
	assert.NoError(t, err)

	host1, err := mn.GenPeer()
	assert.NoError(t, err)
	network1, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host1),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(mock.NewMapDatastore()),
	}...)
	assert.NoError(t, err)

	host2, err := mn.GenPeer()
	assert.NoError(t, err)
	network2, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host2),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(mock.NewMapDatastore()),
	}...)
	assert.NoError(t, err)

	host3, err := mn.GenPeer()
	assert.NoError(t, err)
	network3, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host3),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(mock.NewMapDatastore()),
	}...)
	assert.NoError(t, err)

	assert.NoError(t, mn.LinkAll())
	assert.NoError(t, mn.ConnectAllButSelf())

	testHarness, err := harness.NewTestHarness(harness.DefaultOptions())
	assert.NoError(t, err)

	err = testHarness.GenerateBlocks(10)
	assert.NoError(t, err)

	NewChainService(context.Background(), testHarness.Blockchain().GetBlockByID, testHarness.Blockchain(), network0, testHarness.Blockchain().Params())
	NewChainService(context.Background(), testHarness.Blockchain().GetBlockByID, testHarness.Blockchain(), network1, testHarness.Blockchain().Params())
	NewChainService(context.Background(), testHarness.Blockchain().GetBlockByID, testHarness.Blockchain(), network2, testHarness.Blockchain().Params())
	service := NewChainService(context.Background(), testHarness.Blockchain().GetBlockByID, testHarness.Blockchain(), network3, testHarness.Blockchain().Params())

	sm := NewSyncManager(context.Background(), testHarness.Blockchain(), network2, testHarness.Blockchain().Params(), service)

	ids, err := sm.queryPeers(5)
	assert.NoError(t, err)

	blkID, err := testHarness.Blockchain().GetBlockIDByHeight(5)
	assert.NoError(t, err)

	assert.Len(t, ids, 3)
	assert.Equal(t, blkID, ids[0])
}
