// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"context"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/walletlib"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddrIndex(t *testing.T) {
	ds := mock.NewMapDatastore()

	idx, err := NewAddrIndex(ds, &params.RegestParams)
	assert.NoError(t, err)

	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	err = idx.ConnectBlock(dbtx, params.RegestParams.GenesisBlock)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), idx.outputIndex)
	assert.Equal(t, uint32(0), idx.height)

	note := types.SpendNote{
		ScriptHash: types.NewID(publicAddrScriptHash),
		Amount:     100000,
		AssetID:    types.ID{},
		Salt:       [32]byte{},
		State:      nil,
	}
	salt, err := types.RandomSalt()
	assert.NoError(t, err)
	note.Salt = salt

	state, err := types.RandomSalt()
	assert.NoError(t, err)
	note.State = types.State{state[:]}

	addr, err := walletlib.NewPublicAddressFromCommitment(state[:], &params.RegestParams)
	assert.NoError(t, err)

	commitment, err := note.Commitment()
	assert.NoError(t, err)

	ciphertext, err := note.ToPublicCiphertext()
	assert.NoError(t, err)

	err = idx.ConnectBlock(dbtx, &blocks.Block{
		Header: &blocks.BlockHeader{
			Height: 1,
		},
		Transactions: []*transactions.Transaction{
			transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment: commitment.Bytes(),
						Ciphertext: ciphertext,
					},
				},
			}),
		},
	})
	nullifier, err := types.CalculateNullifier(2, salt, publicAddrScriptCommitment)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, uint64(3), idx.outputIndex)
	assert.Equal(t, uint32(1), idx.height)
	_, ok := idx.nullifiers[nullifier]
	assert.True(t, ok)

	assert.NoError(t, dbtx.Commit(context.Background()))
	assert.NoError(t, idx.flush())

	txs, err := idx.GetTransactionsIDs(ds, addr)
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	err = idx.ConnectBlock(dbtx, &blocks.Block{
		Header: &blocks.BlockHeader{
			Height: 2,
		},
		Transactions: []*transactions.Transaction{
			transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier[:]},
			}),
		},
	})
	assert.NoError(t, dbtx.Commit(context.Background()))
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), idx.outputIndex)
	assert.Equal(t, uint32(2), idx.height)
	_, ok = idx.nullifiers[nullifier]
	assert.False(t, ok)

	txs, err = idx.GetTransactionsIDs(ds, addr)
	assert.NoError(t, err)
	assert.Len(t, txs, 2)

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
}
