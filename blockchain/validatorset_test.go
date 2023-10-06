// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
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
		PeerID:        producerID,
		TotalStake:    20000,
		WeightedStake: 20000,
		Nullifiers: map[types.Nullifier]Stake{
			types.NewNullifier(producerNullifier[:]): {
				Amount:         20000,
				WeightedAmount: 20000,
				Locktime:       time.Unix(0, 0),
				Blockstamp:     time.Now().Add(-time.Hour * 24 * 8),
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
	tx, err := vs.ConnectBlock(blk, 0)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	// Check both validators are committed and return values are correct
	ret1, err := vs.GetValidator(producerID)
	assert.NoError(t, err)
	assert.Equal(t, producerID, ret1.PeerID)
	assert.Equal(t, types.Amount(20000), ret1.TotalStake)
	assert.Equal(t, types.Amount(20000), ret1.WeightedStake)
	stake, ok := ret1.Nullifiers[types.NewNullifier(producerNullifier[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret1.Nullifiers))
	assert.Equal(t, types.Amount(20000), stake.Amount)
	assert.Equal(t, types.Amount(20000), stake.WeightedAmount)
	assert.Equal(t, uint32(1), ret1.EpochBlocks)

	ret2, err := vs.GetValidator(valID)
	assert.NoError(t, err)
	assert.Equal(t, valID, ret2.PeerID)
	assert.Equal(t, types.Amount(100000), ret2.TotalStake)
	assert.Equal(t, types.Amount(100000), ret2.WeightedStake)
	stake, ok = ret2.Nullifiers[types.NewNullifier(nullifier[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret2.Nullifiers))
	assert.Equal(t, types.Amount(100000), stake.Amount)
	assert.Equal(t, types.Amount(100000), stake.WeightedAmount)
	assert.Equal(t, uint32(0), ret2.EpochBlocks)

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
	tx, err = vs.ConnectBlock(blk2, 10000)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	// Check the second validator committed correctly
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, valID2, ret1.PeerID)
	assert.Equal(t, types.Amount(80000), ret1.TotalStake)
	assert.Equal(t, types.Amount(80000), ret1.WeightedStake)
	stake, ok = ret1.Nullifiers[types.NewNullifier(nullifier2[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret1.Nullifiers))
	assert.Equal(t, types.Amount(80000), stake.Amount)
	assert.Equal(t, types.Amount(80000), stake.WeightedAmount)
	assert.Equal(t, uint32(0), ret1.EpochBlocks)

	// Make sure the first validator was removed
	_, err = vs.GetValidator(valID)
	assert.Error(t, err)

	// Check that the unclaimed coins increased.
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(0), ret1.UnclaimedCoins) // Zero unclaimed coins because they were created in this block

	ret2, err = vs.GetValidator(producerID)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(1666), ret2.UnclaimedCoins)

	// Create new block that spends a coinbase in a mint
	blk3 := randomBlock(randomBlockHeader(3, randomID()), 1)
	blk3.Header.Producer_ID = producerIDBytes
	blk3.Header.Timestamp = blk2.Header.Timestamp + (vs.params.EpochLength / 2)
	blk3.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.MintTransaction{
			Nullifiers: [][]byte{producerNullifier[:]},
		}),
	}
	tx, err = vs.ConnectBlock(blk3, 100000)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	// Make sure the producer was removed
	_, err = vs.GetValidator(producerID)
	assert.Error(t, err)

	// Check that the new unclaimed coins were prorated correctly.
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(40000), ret1.UnclaimedCoins)
	assert.Equal(t, types.Amount(80000), ret1.TotalStake)
	assert.Equal(t, types.Amount(80000), ret1.WeightedStake)
	assert.Equal(t, 1, len(ret1.Nullifiers))

	// Test restake
	blk4 := randomBlock(randomBlockHeader(4, randomID()), 1)
	blk4.Header.Producer_ID = producerIDBytes
	blk4.Header.Timestamp = blk.Header.Timestamp + vs.params.EpochLength + 1
	blk4.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.StakeTransaction{
			Validator_ID: valIDBytes2,
			Amount:       80000,
			Nullifier:    nullifier2[:],
		}),
	}
	tx, err = vs.ConnectBlock(blk4, 0)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	// Should be no change in these variables since this is a restake
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(80000), ret1.TotalStake)
	assert.Equal(t, types.Amount(80000), ret1.WeightedStake)
	assert.Equal(t, 1, len(ret1.Nullifiers))

	// Test validator expiration
	blk5 := randomBlock(randomBlockHeader(5, randomID()), 1)
	blk5.Header.Producer_ID = producerIDBytes
	blk5.Header.Timestamp = blk4.Header.Timestamp + int64(ValidatorExpiration.Seconds()) + 1
	tx, err = vs.ConnectBlock(blk5, 100000)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	_, err = vs.GetValidator(valID2)
	assert.Error(t, err)

	// Test weighted stake
	valID3 := randomPeerID()
	valIDBytes3, err := valID3.Marshal()
	assert.NoError(t, err)

	nullifier3 := randomID()
	nullifier4 := randomID()
	blk6 := randomBlock(randomBlockHeader(6, randomID()), 1)
	blk6.Header.Producer_ID = producerIDBytes
	blk6.Header.Timestamp = blk.Header.Timestamp + vs.params.EpochLength + 1
	blk6.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.StakeTransaction{
			Validator_ID: valIDBytes2,
			Amount:       80000,
			Nullifier:    nullifier3[:],
			LockedUntil:  time.Unix(blk6.Header.Timestamp, 0).Add(time.Hour * 24 * 30 * 8).Unix(),
		}),
		transactions.WrapTransaction(&transactions.StakeTransaction{
			Validator_ID: valIDBytes3,
			Amount:       10,
			Nullifier:    nullifier4[:],
			LockedUntil:  0,
		}),
	}
	tx, err = vs.ConnectBlock(blk6, 0)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	// Check the second validator committed correctly
	ret1, err = vs.GetValidator(valID2)
	assert.NoError(t, err)
	assert.Equal(t, valID2, ret1.PeerID)
	assert.Equal(t, types.Amount(80000), ret1.TotalStake)
	assert.Equal(t, types.Amount(83075), ret1.WeightedStake)
	stake, ok = ret1.Nullifiers[types.NewNullifier(nullifier3[:])]
	assert.True(t, ok)
	assert.Equal(t, 1, len(ret1.Nullifiers))
	assert.Equal(t, types.Amount(80000), stake.Amount)
	assert.Equal(t, types.Amount(83075), stake.WeightedAmount)
	assert.Equal(t, uint32(0), ret1.EpochBlocks)

	// Test production over limit

	// First seven blocks should not trigger any penalty as they are within the
	// expected standard deviation.
	for i := uint32(0); i < 7; i++ {
		blk7 := randomBlock(randomBlockHeader(7+i, randomID()), 1)
		blk7.Header.Producer_ID = valIDBytes3
		blk7.Header.Timestamp = blk.Header.Timestamp + vs.params.EpochLength + 1
		tx, err = vs.ConnectBlock(blk7, 0)
		assert.NoError(t, err)
		assert.NoError(t, tx.Commit(FlushRequired))

		ret1, err = vs.GetValidator(valID3)
		assert.NoError(t, err)
		assert.False(t, ret1.CoinbasePenalty)
	}

	// The next two should trigger the coinbase penalty.
	for i := uint32(0); i < 2; i++ {
		blk7 := randomBlock(randomBlockHeader(12+i, randomID()), 1)
		blk7.Header.Producer_ID = valIDBytes3
		blk7.Header.Timestamp = blk.Header.Timestamp + vs.params.EpochLength + 1
		tx, err = vs.ConnectBlock(blk7, 0)
		assert.NoError(t, err)
		assert.NoError(t, tx.Commit(FlushRequired))

		ret1, err = vs.GetValidator(valID3)
		assert.NoError(t, err)
		assert.True(t, ret1.CoinbasePenalty)
		assert.Equal(t, ret1.Strikes, i+1)
	}

	// The next block should cause the nullifier to be banned and the
	// validator to be removed from the validataor set.
	blk7 := randomBlock(randomBlockHeader(15, randomID()), 1)
	blk7.Header.Producer_ID = valIDBytes3
	blk7.Header.Timestamp = blk.Header.Timestamp + vs.params.EpochLength + 1
	tx, err = vs.ConnectBlock(blk7, 0)
	assert.Len(t, tx.NullifiersToBan(), 1)
	assert.Equal(t, types.NewNullifier(nullifier4[:]), tx.NullifiersToBan()[0])
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

	_, err = vs.GetValidator(valID3)
	assert.Error(t, err)
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
	tx, err := vs.ConnectBlock(params.RegestParams.GenesisBlock, 0)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))
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
	tx, err := vs.ConnectBlock(blk, 0)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(FlushRequired))

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
