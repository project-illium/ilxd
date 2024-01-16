// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWalletServerIndex(t *testing.T) {
	ds := mock.NewMapDatastore()

	idx, err := NewWalletServerIndex(ds)
	assert.NoError(t, err)

	// Register view key and make sure it's registered correctly
	viewKey, _, err := icrypto.GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	privKeyBytes, err := crypto.MarshalPrivateKey(viewKey)
	assert.NoError(t, err)

	ul := types.LockingScript{
		ScriptCommitment: types.ID{},
		LockingParams:    make([][]byte, 1),
	}
	ul.LockingParams[0] = make([]byte, 32)
	rand.Read(ul.LockingParams[0][1:])

	err = idx.RegisterViewKey(ds, viewKey, ul.Serialize())
	assert.NoError(t, err)

	_, err = dsFetchIndexValue(ds, &WalletServerIndex{}, walletServerViewKeyPrefix+hex.EncodeToString(privKeyBytes))
	assert.NoError(t, err)

	// Create a block which pays the viewkey and insert it into the index
	note := randSpendNote()
	scriptHash, err := ul.Hash()
	assert.NoError(t, err)
	note.ScriptHash = scriptHash
	commitment, err := note.Commitment()
	assert.NoError(t, err)
	ser, err := note.Serialize()
	assert.NoError(t, err)
	cipherText, err := viewKey.GetPublic().(*icrypto.Curve25519PublicKey).Encrypt(ser)
	assert.NoError(t, err)

	sub := idx.Subscribe()

	blk := &blocks.Block{
		Header: &blocks.BlockHeader{
			Height: 1,
		},
		Transactions: []*transactions.Transaction{
			transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment: commitment[:],
						Ciphertext: cipherText,
					},
				},
			}),
		},
	}

	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	err = idx.ConnectBlock(dbtx, blk)
	assert.NoError(t, err)
	assert.NoError(t, dbtx.Commit(context.Background()))

	select {
	case <-sub.C:
	case <-time.After(time.Second * 10):
		t.Errorf("failed to detect subscription")
	}

	// Check that the tx was recorded correctly
	txids, err := idx.GetTransactionsIDs(ds, viewKey)
	assert.NoError(t, err)
	assert.Len(t, txids, 1)

	// Check that we can fetch the proof correctly and that it validates
	proofs, merkleRoot, err := idx.GetTxoProofs([]types.ID{commitment})
	assert.NoError(t, err)
	assert.Len(t, proofs, 1)

	valid, err := blockchain.ValidateInclusionProof(commitment.Bytes(), proofs[0].Index, proofs[0].Hashes, proofs[0].Flags, merkleRoot[:])
	assert.NoError(t, err)
	assert.True(t, valid)

	// Close the index, then repo and make sure it loads state correctly
	sub.Close()
	err = idx.Close(ds)
	assert.NoError(t, err)

	idx, err = NewWalletServerIndex(ds)
	assert.NoError(t, err)

	assert.Len(t, idx.nullifiers, 1)
	assert.Equal(t, idx.bestBlockHeight, uint32(1))
	assert.NotEqual(t, make([]byte, 32), idx.bestBlockID[:])

	sub = idx.Subscribe()

	// Create a block spending the utxo and make sure the tx is recorded
	nullifier, err := types.CalculateNullifier(proofs[0].Index, note.Salt, ul.ScriptCommitment.Bytes(), ul.LockingParams...)
	assert.NoError(t, err)

	blk2 := &blocks.Block{
		Header: &blocks.BlockHeader{
			Height: 2,
		},
		Transactions: []*transactions.Transaction{
			transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier[:]},
			}),
		},
	}

	dbtx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	err = idx.ConnectBlock(dbtx, blk2)
	assert.NoError(t, err)
	assert.NoError(t, dbtx.Commit(context.Background()))

	select {
	case <-sub.C:
	case <-time.After(time.Second * 10):
		t.Errorf("failed to detect subscription")
	}

	txids, err = idx.GetTransactionsIDs(ds, viewKey.(*icrypto.Curve25519PrivateKey))
	assert.NoError(t, err)
	assert.Len(t, txids, 2)

	// Make sure the indexer deleted the appropriate data after the spend
	proofs, merkleRoot, err = idx.GetTxoProofs([]types.ID{commitment})
	assert.Error(t, err)
	assert.Len(t, proofs, 0)
	assert.Len(t, idx.nullifiers, 0)

	_, err = dsFetchIndexValue(ds, idx, walletServerNullifierKeyPrefix+string(privKeyBytes)+"/"+nullifier.String())
	assert.Error(t, err)
	sub.Close()

	// Test rescanning
	ds = mock.NewMapDatastore()
	idx, err = NewWalletServerIndex(ds)
	assert.NoError(t, err)
	idx.bestBlockHeight = 2

	assert.NoError(t, idx.RegisterViewKey(ds, viewKey, ul.Serialize()))
	err = idx.RescanViewkey(ds, viewKey, nil, 0, func(height uint32) (*blocks.Block, error) {
		if height == 0 {
			return &blocks.Block{
				Header:       &blocks.BlockHeader{Height: 0},
				Transactions: nil,
			}, nil
		} else if height == 1 {
			return blk, nil
		}
		return blk2, nil
	})
	assert.NoError(t, err)

	txids, err = idx.GetTransactionsIDs(ds, viewKey)
	assert.NoError(t, err)
	assert.Len(t, txids, 2)
}

func randSpendNote() types.SpendNote {
	note := types.SpendNote{
		ScriptHash: types.ID{},
		Amount:     20000,
		AssetID:    types.ID{},
		State:      types.State{},
		Salt:       [32]byte{},
	}
	scriptHash, _ := types.RandomSalt()
	copy(note.ScriptHash[:], scriptHash[:])
	assetID, _ := types.RandomSalt()
	copy(note.AssetID[:], assetID[:])
	salt, _ := types.RandomSalt()
	note.Salt = salt
	return note
}
