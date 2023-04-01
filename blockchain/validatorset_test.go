// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"math"
	mrand "math/rand"
	"testing"
	"time"
)

func TestValidatorSet_CommitBlock(t *testing.T) {
	ds := mock.NewMapDatastore()
	vs := NewValidatorSet(&params.RegestParams, ds)

	// Let's add a block producer so we can check epoch
	// blocks returns correctly.
	producerID := randomPeerID()
	producerNullifier := randomID()
	producerIDBytes, err := producerID.Marshal()
	assert.NoError(t, err)
	validator1 := &Validator{
		PeerID:     producerID,
		TotalStake: 20000,
		Nullifiers: map[types.Nullifier]Stake{
			types.NewNullifier(producerNullifier[:]): {
				Amount:     20000,
				Blockstamp: time.Now(),
			},
		},
	}
	vs.validators[producerID] = validator1
	vs.nullifierMap[types.NewNullifier(producerNullifier[:])] = validator1

	// Now commit a block that creates a new validator
	valID := randomPeerID()
	valIDBytes, err := valID.Marshal()
	assert.NoError(t, err)
	nullifier := randomID()
	blk := randomBlock(randomBlockHeader(1, randomID()), 1)
	blk.Header.Producer_ID = producerIDBytes
	blk.Header.Timestamp = time.Now().Unix()
	blk.Transactions[0] = transactions.WrapTransaction(&transactions.StakeTransaction{
		Validator_ID: valIDBytes,
		Amount:       100000,
		Nullifier:    nullifier[:],
	})
	assert.NoError(t, vs.CommitBlock(blk, 0, flushRequired))

	// Check both validators are committed and return values are correct
	ret1, err := vs.GetValidator(producerID)
	assert.NoError(t, err)
	assert.Equal(t, producerID, ret1.PeerID)
	assert.Equal(t, types.Amount(20000), ret1.TotalStake)
	stake, ok := ret1.Nullifiers[types.NewNullifier(producerNullifier[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret1.Nullifiers))
	assert.Equal(t, types.Amount(20000), stake.Amount)
	assert.Equal(t, uint32(1), ret1.epochBlocks)

	ret2, err := vs.GetValidator(valID)
	assert.NoError(t, err)
	assert.Equal(t, valID, ret2.PeerID)
	assert.Equal(t, types.Amount(100000), ret2.TotalStake)
	stake, ok = ret2.Nullifiers[types.NewNullifier(nullifier[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret2.Nullifiers))
	assert.Equal(t, types.Amount(100000), stake.Amount)
	assert.Equal(t, uint32(0), ret2.epochBlocks)

	// Check nullifiers are in the nullifier map
	_, ok = vs.nullifierMap[types.NewNullifier(producerNullifier[:])]
	assert.True(t, ok)
	_, ok = vs.nullifierMap[types.NewNullifier(nullifier[:])]
	assert.True(t, ok)

	// Now commit a block that spends the nullifier and creates a new one
	// This block will also increase the validator rewards as well.
	valID2 := randomPeerID()
	valIDBytes2, err := valID2.Marshal()
	assert.NoError(t, err)
	nullifier2 := randomID()
	blk2 := randomBlock(randomBlockHeader(2, randomID()), 1)
	blk2.Header.Producer_ID = producerIDBytes
	blk2.Header.Timestamp = blk.Header.Timestamp + vs.params.EpochLength + 1
	blk2.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.StakeTransaction{
			Validator_ID: valIDBytes2,
			Amount:       80000,
			Nullifier:    nullifier2[:],
		}),
		transactions.WrapTransaction(&transactions.StandardTransaction{
			Nullifiers: [][]byte{nullifier[:]},
		}),
	}
	assert.NoError(t, vs.CommitBlock(blk2, 10000, flushRequired))

	// Check the second validator committed correctly
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, valID2, ret1.PeerID)
	assert.Equal(t, types.Amount(80000), ret1.TotalStake)
	stake, ok = ret1.Nullifiers[types.NewNullifier(nullifier2[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret1.Nullifiers))
	assert.Equal(t, types.Amount(80000), stake.Amount)
	assert.Equal(t, uint32(0), ret1.epochBlocks)

	// Make sure the first validator was removed
	_, err = vs.GetValidator(valID)
	assert.Error(t, err)

	// Check that the unclaimed coins increased.
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(0), ret1.unclaimedCoins) // Zero unclaimed coins because they were created in this block

	ret2, err = vs.GetValidator(producerID)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(2000), ret2.unclaimedCoins)

	// Create new block that spends a coinbase and a mint
	blk3 := randomBlock(randomBlockHeader(3, randomID()), 1)
	blk3.Header.Producer_ID = producerIDBytes
	blk3.Header.Timestamp = blk2.Header.Timestamp + (vs.params.EpochLength / 2)
	blk3.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.MintTransaction{
			Nullifiers: [][]byte{producerNullifier[:]},
		}),
	}
	assert.NoError(t, vs.CommitBlock(blk3, 100000, flushRequired))

	// Make sure the producer was removed
	_, err = vs.GetValidator(producerID)
	assert.Error(t, err)

	// Check that the new unclaimed coins were prorated correctly.
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(50000), ret1.unclaimedCoins)
}

func TestValidatorSet_Init(t *testing.T) {
	ds := mock.NewMapDatastore()
	err := populateDatabase(ds, 5000)
	assert.NoError(t, err)

	index := NewBlockIndex(ds)
	err = index.Init()
	assert.NoError(t, err)

	// Init with empty consistency state
	vs := NewValidatorSet(&params.RegestParams, mock.NewMapDatastore())
	assert.NoError(t, vs.Init(index.Tip()))

	// Init with flush height at genesis
	vs = NewValidatorSet(&params.RegestParams, mock.NewMapDatastore())
	assert.NoError(t, vs.CommitBlock(params.RegestParams.GenesisBlock, 0, flushRequired))
	assert.NoError(t, vs.Init(index.Tip()))

	// Set status to flush ongoing and re-init
	assert.NoError(t, dsPutValidatorSetConsistencyStatus(vs.ds, scsFlushOngoing))
	assert.NoError(t, vs.Init(index.Tip()))
}

func TestValidatorSetMethods(t *testing.T) {
	ds := mock.NewMapDatastore()
	vs := NewValidatorSet(&params.RegestParams, ds)

	// Commit a block that creates a new validator
	valID := randomPeerID()
	valIDBytes, err := valID.Marshal()
	assert.NoError(t, err)
	nullifier := randomID()
	blk := randomBlock(randomBlockHeader(1, randomID()), 1)
	blk.Transactions[0] = transactions.WrapTransaction(&transactions.StakeTransaction{
		Validator_ID: valIDBytes,
		Amount:       100000,
		Nullifier:    nullifier[:],
	})
	assert.NoError(t, vs.CommitBlock(blk, 0, flushRequired))

	ret, err := vs.GetValidator(valID)
	assert.NoError(t, err)
	assert.Equal(t, valID, ret.PeerID)

	assert.True(t, vs.ValidatorExists(valID))
	assert.True(t, vs.NullifierExists(types.NewNullifier(nullifier[:])))

	val2ID := randomPeerID()
	null2 := randomID()
	_, err = vs.GetValidator(val2ID)
	assert.Error(t, err)
	assert.False(t, vs.ValidatorExists(val2ID))
	assert.False(t, vs.NullifierExists(types.NewNullifier(null2[:])))

	assert.Equal(t, types.Amount(100000), vs.TotalStaked())

	assert.Equal(t, valID, vs.WeightedRandomValidator())
}

func TestNewValidatorSet(t *testing.T) {
	epochLen := float64(60 * 60 * 24 * 7)
	stakePercentage := float64(.15)
	mean, sigma := calcStdDeviation(epochLen, stakePercentage)
	fmt.Println(mean, mean+(sigma*7))
	fmt.Println(blockProductionLimit(epochLen, stakePercentage))
}

func calcStdDeviation(epochLen float64, stakePercent float64) (float64, float64) {
	stakePercent *= 100
	avgs := make([]int, 1000)
	for r := 0; r < 1000; r++ {
		blks := 0
		for i := 0; i < int(epochLen); i++ {
			x := mrand.Intn(1000)
			if float64(x) < stakePercent*10 {
				blks++
			}
		}
		avgs[r] = blks
	}
	total := 0
	for _, avg := range avgs {
		total += avg
	}
	mean := float64(total) / 1000

	deviation2total := float64(0)
	for _, avg := range avgs {
		deviation2total += math.Pow(float64(avg)-mean, 2)
	}

	variance := deviation2total / 1000

	return mean, math.Sqrt(variance)
}
