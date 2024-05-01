// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"crypto/rand"
	"github.com/go-test/deep"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPutGetHeader(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutHeader(dbtx, header))
	assert.NoError(t, dbtx.Commit(context.Background()))

	header2, err := dsFetchHeader(ds, header.ID())
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(header, header2))
}

func TestPutGetDeleteBlock(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	block := randomBlock(header, 5)
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutBlock(dbtx, block))
	assert.NoError(t, dbtx.Commit(context.Background()))

	exists, err := dsBlockExists(ds, block.ID())
	assert.NoError(t, err)
	assert.True(t, exists)

	block2, err := dsFetchBlock(ds, block.ID())
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(block, block2))

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteBlock(dbtx, block.ID()))
	assert.NoError(t, dbtx.Commit(context.Background()))

	block2, err = dsFetchBlock(ds, block.ID())
	assert.Error(t, err)
	assert.Nil(t, block2)
}

func TestPutGetBlockIDByHeight(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutBlockIDFromHeight(dbtx, header.ID(), 5))
	assert.NoError(t, dbtx.Commit(context.Background()))

	id, err := dsFetchBlockIDFromHeight(ds, 5)
	assert.NoError(t, err)
	assert.Equal(t, header.ID(), id)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteBlockIDFromHeight(dbtx, 5))
	assert.NoError(t, dbtx.Commit(context.Background()))

	_, err = dsFetchBlockIDFromHeight(ds, 5)
	assert.Error(t, err)
}

func TestPutGetDeleteBlockIndexState(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	block := randomBlock(header, 5)
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutBlockIndexState(dbtx, &blockNode{
		ds:      ds,
		blockID: block.ID(),
		height:  block.Header.Height,
	}))
	assert.NoError(t, dbtx.Commit(context.Background()))

	state, err := dsFetchBlockIndexState(ds)
	assert.NoError(t, err)
	assert.Equal(t, block.ID(), state.blockID)
	assert.Equal(t, block.Header.Height, state.height)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteBlockIndexState(dbtx))
	assert.NoError(t, dbtx.Commit(context.Background()))
	_, err = dsFetchBlockIndexState(ds)
	assert.Error(t, err)
}

func TestPutGetValidatorSetConsistencyStatus(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutValidatorSetConsistencyStatus(ds, scsFlushOngoing))
	assert.NoError(t, dbtx.Commit(context.Background()))

	status, err := dsFetchValidatorSetConsistencyStatus(ds)
	assert.NoError(t, err)
	assert.Equal(t, scsFlushOngoing, status)
}

func TestPutGetValidatorSetLastFlushHeight(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutValidatorLastFlushHeight(dbtx, 100))
	assert.NoError(t, dbtx.Commit(context.Background()))

	height, err := dsFetchValidatorLastFlushHeight(ds)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), height)
}

func TestPutGetDeleteValidator(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	id := randomID()
	peer := randomPeerID()
	validator := &Validator{
		PeerID:        peer,
		TotalStake:    100,
		WeightedStake: 200,
		Nullifiers: map[types.Nullifier]Stake{
			types.NewNullifier(id[:]): {
				Amount:         100,
				WeightedAmount: 200,
				Locktime:       time.Now(),
				Blockstamp:     time.Now(),
			},
		},
		UnclaimedCoins:  50,
		EpochBlocks:     17,
		Strikes:         3,
		CoinbasePenalty: true,
		ExpectedBlocks:  14,
	}
	assert.NoError(t, dsPutValidator(dbtx, validator))
	assert.NoError(t, dbtx.Commit(context.Background()))

	validators, err := dsFetchValidators(ds)
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal([]*Validator{validator}, validators))

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteValidator(dbtx, peer))
	assert.NoError(t, dbtx.Commit(context.Background()))
	validators, err = dsFetchValidators(ds)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(validators))

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutValidator(dbtx, validator))
	assert.NoError(t, dbtx.Commit(context.Background()))
	validators, err = dsFetchValidators(ds)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(validators))

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteValidatorSet(dbtx))
	assert.NoError(t, dbtx.Commit(context.Background()))
	validators, err = dsFetchValidators(ds)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(validators))
}

func TestPutExistsDeleteNullifiers(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	n1, n2, n3 := randomID(), randomID(), randomID()
	assert.NoError(t, dsPutNullifiers(dbtx, []types.Nullifier{types.NewNullifier(n1[:]), types.NewNullifier(n2[:])}))
	assert.NoError(t, dbtx.Commit(context.Background()))

	exists, err := dsNullifierExists(ds, types.NewNullifier(n1[:]))
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dsNullifierExists(ds, types.NewNullifier(n2[:]))
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dsNullifierExists(ds, types.NewNullifier(n3[:]))
	assert.NoError(t, err)
	assert.False(t, exists)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteNullifierSet(dbtx))
	assert.NoError(t, dbtx.Commit(context.Background()))
	exists, err = dsNullifierExists(ds, types.NewNullifier(n2[:]))
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestPutExistsTxoRoot(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	n1, n2, n3 := randomID(), randomID(), randomID()
	assert.NoError(t, dsPutTxoSetRoot(dbtx, n1))
	assert.NoError(t, dsPutTxoSetRoot(dbtx, n2))
	assert.NoError(t, dbtx.Commit(context.Background()))

	exists, err := dsTxoSetRootExists(ds, n1)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dsTxoSetRootExists(ds, n2)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dsTxoSetRootExists(ds, n3)
	assert.NoError(t, err)
	assert.False(t, exists)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteTxoRootSet(dbtx))
	assert.NoError(t, dbtx.Commit(context.Background()))
	exists, err = dsTxoSetRootExists(ds, n2)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDebitCreditBalanceTreasury(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	assert.NoError(t, dsInitTreasury(ds))

	balance, err := dsFetchTreasuryBalance(ds)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(0), balance)

	assert.NoError(t, dsCreditTreasury(dbtx, 999))
	assert.NoError(t, dbtx.Commit(context.Background()))

	balance, err = dsFetchTreasuryBalance(ds)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(999), balance)

	assert.NoError(t, dsDebitTreasury(dbtx, 100))
	assert.NoError(t, dbtx.Commit(context.Background()))

	balance, err = dsFetchTreasuryBalance(ds)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(899), balance)
}

func TestPutFetchDeleteAccumulator(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	acc := NewAccumulator()
	id := randomID()
	acc.Insert(id[:], true)
	id = randomID()
	acc.Insert(id[:], true)
	assert.NoError(t, dsPutAccumulator(dbtx, acc))
	assert.NoError(t, dbtx.Commit(context.Background()))

	acc2, err := dsFetchAccumulator(ds)
	assert.NoError(t, err)

	assert.Empty(t, deep.Equal(acc, acc2))
	assert.Equal(t, acc.Root(), acc2.Root())

	for k, v := range acc2.proofs {
		assert.Equal(t, v, acc.proofs[k])
	}
	for k, v := range acc2.lookupMap {
		assert.Equal(t, v, acc.lookupMap[k])
	}
	assert.Len(t, acc2.proofs, len(acc.proofs))
	assert.Len(t, acc2.lookupMap, len(acc.lookupMap))

	assert.NoError(t, dsPutAccumulatorCheckpoint(dbtx, 50000, acc))
	assert.NoError(t, dbtx.Commit(context.Background()))

	acc2, err = dsFetchAccumulatorCheckpoint(ds, 50000)
	assert.NoError(t, err)

	assert.Empty(t, deep.Equal(acc, acc2))
	assert.Equal(t, acc.Root(), acc2.Root())

	for k, v := range acc2.proofs {
		assert.Equal(t, v, acc.proofs[k])
	}
	for k, v := range acc2.lookupMap {
		assert.Equal(t, v, acc.lookupMap[k])
	}
	assert.Len(t, acc2.proofs, len(acc.proofs))
	assert.Len(t, acc2.lookupMap, len(acc.lookupMap))

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteAccumulator(dbtx))
	assert.NoError(t, dbtx.Commit(context.Background()))
	_, err = dsFetchAccumulator(ds)
	assert.Error(t, err)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsDeleteAccumulatorCheckpoints(dbtx))
	assert.NoError(t, dbtx.Commit(context.Background()))
	_, err = dsFetchAccumulatorCheckpoint(ds, 50000)
	assert.Error(t, err)
}

func TestPutGetAccumulatorConsistencyStatus(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutAccumulatorConsistencyStatus(ds, scsFlushOngoing))
	assert.NoError(t, dbtx.Commit(context.Background()))

	status, err := dsFetchAccumulatorConsistencyStatus(ds)
	assert.NoError(t, err)
	assert.Equal(t, scsFlushOngoing, status)
}

func TestPutGetAccumulatorLastFlushHeight(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutAccumulatorLastFlushHeight(dbtx, 100))
	assert.NoError(t, dbtx.Commit(context.Background()))

	height, err := dsFetchAccumulatorLastFlushHeight(ds)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), height)
}

func TestIncrementFetchCurrentSupply(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	assert.NoError(t, dsInitCurrentSupply(ds))

	supply, err := dsFetchCurrentSupply(dbtx)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(0), supply)

	assert.NoError(t, dsIncrementCurrentSupply(dbtx, 999))
	assert.NoError(t, dbtx.Commit(context.Background()))

	supply, err = dsFetchCurrentSupply(dbtx)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(999), supply)
}

func TestPutGetEpoch(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	r := make([]byte, 32)
	rand.Read(r[:])
	assert.NoError(t, dsPutEpoch(dbtx, types.NewID(r), 5))
	assert.NoError(t, dbtx.Commit(context.Background()))

	id, height, err := dsFetchEpoch(ds)
	assert.NoError(t, err)
	assert.Equal(t, r, id.Bytes())
	assert.Equal(t, uint32(5), height)
}
