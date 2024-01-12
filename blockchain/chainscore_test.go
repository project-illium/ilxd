// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/harness"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/zk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBlockchain_CalcChainScore(t *testing.T) {
	testHarness, err := harness.NewTestHarness(harness.DefaultOptions())
	assert.NoError(t, err)

	assert.NoError(t, testHarness.GenerateBlocks(10))

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(testHarness.Blockchain().Params()), blockchain.Verifier(&zk.MockVerifier{}))
	assert.NoError(t, err)

	for i := uint32(1); i < 5; i++ {
		blk, err := testHarness.Blockchain().GetBlockByHeight(i)
		assert.NoError(t, err)

		assert.NoError(t, chain.ConnectBlock(blk, blockchain.BFNone))
	}

	blks := make([]*blocks.Block, 0, 5)
	for i := uint32(50); i < 10; i++ {
		blk, err := testHarness.Blockchain().GetBlockByHeight(i)
		assert.NoError(t, err)
		blks = append(blks, blk)
	}

	score, err := chain.CalcChainScore(blks)
	assert.NoError(t, err)

	assert.Equal(t, blockchain.ChainScore(0), score)
}
