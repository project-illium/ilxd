// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"context"
	"github.com/go-test/deep"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/blockchain/harness"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChainService(t *testing.T) {
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
		net.MaxMessageSize(repo.DefaultMaxMessageSize),
	}...)
	assert.NoError(t, err)

	testHarness1, err := harness.NewTestHarness(harness.DefaultOptions())
	assert.NoError(t, err)

	err = testHarness1.GenerateBlocks(10)
	assert.NoError(t, err)

	service1 := NewChainService(context.Background(), testHarness1.Blockchain().GetBlockByID, testHarness1.Blockchain(), network1, testHarness1.Blockchain().Params())

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
		net.MaxMessageSize(repo.DefaultMaxMessageSize),
	}...)
	assert.NoError(t, err)

	testHarness2, err := harness.NewTestHarness(harness.DefaultOptions())
	assert.NoError(t, err)

	err = testHarness2.GenerateBlocks(10)
	assert.NoError(t, err)

	service2 := NewChainService(context.Background(), testHarness2.Blockchain().GetBlockByID, testHarness2.Blockchain(), network2, testHarness2.Blockchain().Params())

	assert.NoError(t, mn.LinkAll())
	assert.NoError(t, mn.ConnectAllButSelf())

	b5, err := testHarness1.Blockchain().GetBlockByHeight(5)
	assert.NoError(t, err)

	ret, err := service2.GetBlockTxids(host1.ID(), b5.ID())
	assert.NoError(t, err)
	assert.Equal(t, b5.Txids(), ret)

	b4, err := testHarness2.Blockchain().GetBlockByHeight(4)
	assert.NoError(t, err)

	ret, err = service1.GetBlockTxids(host2.ID(), b4.ID())
	assert.NoError(t, err)
	assert.Equal(t, b4.Txids(), ret)

	ret2, err := service2.GetBlockTxs(host1.ID(), b5.ID(), []uint32{0})
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(b5.GetTransactions(), ret2))

	ret2, err = service1.GetBlockTxs(host2.ID(), b4.ID(), []uint32{0})
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(b4.GetTransactions(), ret2))

	ret3, err := service2.GetBlock(host1.ID(), b5.ID())
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(b5, ret3))

	ret3, err = service1.GetBlock(host2.ID(), b4.ID())
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(b4, ret3))

	retID, err := service1.GetBlockID(host2.ID(), b4.Header.Height)
	assert.NoError(t, err)
	assert.Equal(t, b4.Header.ID(), retID)

	bestID, bestHeight, err := service1.GetBest(host2.ID())
	assert.NoError(t, err)
	b11, h11, _ := testHarness2.Blockchain().BestBlock()
	assert.Equal(t, h11, bestHeight)
	assert.Equal(t, b11, bestID)

	stream, err := service1.GetHeadersStream(host2.ID(), 0)
	assert.NoError(t, err)
	expected := uint32(0)
	for h := range stream {
		assert.Equal(t, expected, h.Height)
		expected++
	}
	assert.Equal(t, uint32(11), expected)

	stream2, err := service1.GetBlockTxsStream(host2.ID(), 0)
	assert.NoError(t, err)
	i := uint32(0)
	for txs := range stream2 {
		blk, err := testHarness2.Blockchain().GetBlockByHeight(i)
		assert.NoError(t, err)
		for x, tx := range blk.Transactions {
			assert.Equal(t, tx.ID(), txs.Transactions[x].ID())
		}
		i++
	}
	assert.Equal(t, uint32(11), i)
}
