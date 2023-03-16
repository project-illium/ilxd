// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
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

func TestPutGetBlock(t *testing.T) {
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
}

func TestPutGetBlockIndexState(t *testing.T) {
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
		PeerID:     peer,
		TotalStake: 100,
		Nullifiers: map[types.Nullifier]Stake{
			types.NewNullifier(id[:]): {
				Amount:     100,
				Blockstamp: time.Now(),
			},
		},
		unclaimedCoins: 50,
		epochBlocks:    17,
		dirty:          true,
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
	validators, err = dsFetchValidators(ds)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(validators))

}

func TestPutExistsNullifiers(t *testing.T) {
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
}

func TestDebitCreditBalanceTreasury(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	assert.NoError(t, dsInitTreasury(ds))

	balance, err := dsFetchTreasuryBalance(ds)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), balance)

	assert.NoError(t, dsCreditTreasury(dbtx, 999))
	assert.NoError(t, dbtx.Commit(context.Background()))

	balance, err = dsFetchTreasuryBalance(ds)
	assert.NoError(t, err)
	assert.Equal(t, uint64(999), balance)

	assert.NoError(t, dsDebitTreasury(dbtx, 100))
	assert.NoError(t, dbtx.Commit(context.Background()))

	balance, err = dsFetchTreasuryBalance(ds)
	assert.NoError(t, err)
	assert.Equal(t, uint64(899), balance)
}

func TestPutFetchAccumulator(t *testing.T) {
	ds := mock.NewMapDatastore()
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	acc := NewAccumulator()
	id := randomID()
	acc.Insert(id[:], true)
	assert.NoError(t, dsPutAccumulator(dbtx, acc))
	assert.NoError(t, dbtx.Commit(context.Background()))

	acc2, err := dsFetchAccumulator(ds)
	assert.NoError(t, err)

	assert.Empty(t, deep.Equal(acc, acc2))
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
	assert.Equal(t, uint64(0), supply)

	assert.NoError(t, dsIncrementCurrentSupply(dbtx, 999))
	assert.NoError(t, dbtx.Commit(context.Background()))

	supply, err = dsFetchCurrentSupply(dbtx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(999), supply)
}
