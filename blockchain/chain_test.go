// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"math"
	"testing"
)

func TestBlockchain(t *testing.T) {
	verifier := &zk.MockVerifier{}
	verifier.SetValid(true)
	b, err := NewBlockchain(DefaultOptions(), Verifier(verifier))
	assert.NoError(t, err)

	// Chain should have genesis block
	genesis, err := b.GetBlockByID(params.RegestParams.GenesisBlock.ID())
	assert.NoError(t, err)
	genesisID := genesis.ID()
	assert.Equal(t, params.RegestParams.GenesisBlock.ID(), genesisID)

	genesis, err = b.GetBlockByHeight(0)
	assert.NoError(t, err)
	assert.Equal(t, params.RegestParams.GenesisBlock.ID(), genesisID)

	// Db should have currently supply set to genesis coins
	dbtx, err := b.ds.NewTransaction(context.Background(), true)
	assert.NoError(t, err)
	nCoins, err := dsFetchCurrentSupply(dbtx)
	assert.NoError(t, err)
	assert.NoError(t, dbtx.Commit(context.Background()))
	assert.Equal(t, types.Amount(params.RegestParams.GenesisBlock.Transactions[0].GetCoinbaseTransaction().NewCoins), nCoins)

	// The validator set should have 1 validator in it
	assert.Equal(t, 1, len(b.validatorSet.validators))
	assert.Equal(t, types.Amount(params.RegestParams.GenesisBlock.Transactions[1].GetStakeTransaction().Amount), b.validatorSet.TotalStaked())

	// Try connecting the genesis block again and make sure it raises a dup block error
	assert.Error(t, b.ConnectBlock(b.params.GenesisBlock, BFGenesisValidation))

	// Test treasury withdrawal
	blk := &blocks.Block{
		Header: &blocks.BlockHeader{
			Version:   1,
			Height:    1,
			Parent:    genesisID[:],
			Timestamp: genesis.Header.Timestamp + 1,
		},
		Transactions: []*transactions.Transaction{
			transactions.WrapTransaction(&transactions.TreasuryTransaction{
				Amount: 10000,
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
					},
				},
				ProposalHash: make([]byte, MaxDocumentHashLen),
				Proof:        make([]byte, 11000),
			}),
		},
	}
	// Withdrawal exceeds treasury balance
	validatorKey, err := crypto.UnmarshalPrivateKey(params.RegtestGenesisKey)
	assert.NoError(t, err)
	validatorID, err := peer.IDFromPrivateKey(validatorKey)
	assert.NoError(t, err)
	assert.NoError(t, finalizeAndSignBlock(blk, validatorKey))
	assert.Error(t, b.ConnectBlock(blk, BFNone))

	// Withdrawal within treasury balance
	dbtx, err = b.ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsCreditTreasury(dbtx, 20000))
	assert.NoError(t, dbtx.Commit(context.Background()))
	assert.NoError(t, b.ConnectBlock(blk, BFNone))

	// Test change of epoch
	blk2 := proto.Clone(blk).(*blocks.Block)
	blk2.Header.Height = blk.Header.Height + 1
	prev := blk.Header.ID()
	blk2.Header.Parent = prev[:]
	blk2.Header.Timestamp = params.RegestParams.GenesisBlock.Header.Timestamp + params.RegestParams.EpochLength + 1
	assert.NoError(t, finalizeAndSignBlock(blk2, validatorKey))
	assert.NoError(t, b.ConnectBlock(blk2, BFNone))

	dbtx, err = b.ds.NewTransaction(context.Background(), true)
	assert.NoError(t, err)
	supply, err := dsFetchCurrentSupply(dbtx)
	assert.NoError(t, err)
	assert.Greater(t, supply, params.RegestParams.GenesisBlock.Transactions[0].GetCoinbaseTransaction().NewCoins)
	assert.NoError(t, dbtx.Commit(context.Background()))
	treasuryBal, err := dsFetchTreasuryBalance(b.ds)
	assert.NoError(t, err)
	assert.Greater(t, treasuryBal, uint64(0))

	val, err := b.validatorSet.GetValidator(validatorID)
	assert.NoError(t, err)
	assert.Greater(t, val.UnclaimedCoins, uint64(0))
}

func TestCalculateNextCoinbaseDistribution(t *testing.T) {
	var prevCoinbase, total types.Amount
	for i := int64(0); i < params.MainnetParams.InitialDistributionPeriods; i++ {
		coinbase := calculateNextCoinbaseDistribution(&params.MainnetParams, i)
		if i > 0 {
			assert.Less(t, coinbase, prevCoinbase)
		}
		prevCoinbase = coinbase
		total += coinbase
	}
	initalCoins := types.Amount(params.MainnetParams.GenesisBlock.Transactions[0].GetCoinbaseTransaction().NewCoins)
	// The algorithm doesn't hit the nail on the head perfectly in terms of distributing
	// the target supply in the initial distribution periods. But let's just check make
	// sure it's within some tolerable amount. In this case around three tenths of
	// a percent.
	assert.Less(t, (float64(total+initalCoins)-float64(params.MainnetParams.TargetDistribution))/float64(params.MainnetParams.TargetDistribution), float64(.0035))

	prevCoinbase, total = 0, 0
	for i := params.MainnetParams.InitialDistributionPeriods + 1; i < params.MainnetParams.InitialDistributionPeriods+53; i++ {
		coinbase := calculateNextCoinbaseDistribution(&params.MainnetParams, i)
		assert.Greater(t, coinbase, prevCoinbase)
		prevCoinbase = coinbase
		total += coinbase
	}

	// Same here. Make sure the long term inflation rate is within some tolerable margin of 2%.
	assert.Less(t, (float64(total)/float64(params.MainnetParams.TargetDistribution))-float64(.02), float64(.00001))
}

func TestCalculateNextValidatorReward(t *testing.T) {
	for i := int64(0); i < params.MainnetParams.InitialDistributionPeriods; i++ {
		coinbase := calculateNextCoinbaseDistribution(&params.MainnetParams, i)
		validatorReward := calculateNextValidatorReward(&params.MainnetParams, i)
		assert.Equal(t, math.Round(float64(validatorReward)/float64(coinbase)*100)/100, (float64(100)-params.MainnetParams.TreasuryPercentage)/100)
	}
}

func finalizeAndSignBlock(blk *blocks.Block, privKey crypto.PrivKey) error {
	merkleRoot := TransactionsMerkleRoot(blk.Transactions)
	blk.Header.TxRoot = merkleRoot[:]

	id, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return err
	}
	pidBytes, err := id.Marshal()
	if err != nil {
		return err
	}
	blk.Header.Producer_ID = pidBytes
	sigHash, err := blk.Header.SigHash()
	if err != nil {
		return err
	}
	sig, err := privKey.Sign(sigHash)
	if err != nil {
		return err
	}
	blk.Header.Signature = sig
	return nil
}

/*
	n := float64(1<<60) * 0.8 // total number of coins

	periods := float64(520)
	w_0 := (n / periods) * 4.9
	w_520 := (1 << 60) * 0.000379

	r := math.Pow((w_520 / w_0), (1 / periods))

	total := float64(0)
	for i := float64(0); i < periods; i++ {
		w_i := w_0 * math.Pow(r, i)
		total += w_i
		fmt.Println(w_i)
	}
	fmt.Println()
	fmt.Println(w_520)
	fmt.Println(total / (1 << 60) * .8)
*/
