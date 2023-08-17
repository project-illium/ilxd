// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/libp2p/go-libp2p/core/crypto"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/stretchr/testify/assert"
	"testing"
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

	err = idx.RegisterViewKey(ds, viewKey.(*icrypto.Curve25519PrivateKey))
	assert.NoError(t, err)

	_, err = dsFetchIndexValue(ds, &WalletServerIndex{}, walletServerViewKeyPrefix+hex.EncodeToString(privKeyBytes))
	assert.NoError(t, err)

	// Create a block which pays the viewkey and insert it into the index
	ul := types.UnlockingScript{
		ScriptCommitment: make([]byte, 32),
		ScriptParams:     make([][]byte, 1),
	}
	ul.ScriptParams[0] = make([]byte, 32)
	rand.Read(ul.ScriptCommitment)
	rand.Read(ul.ScriptParams[0])

	note := randSpendNote()
	note.ScriptHash = ul.Hash().Bytes()
	commitment := note.Commitment()
	cipherText, err := viewKey.GetPublic().(*icrypto.Curve25519PublicKey).Encrypt(note.Serialize())
	assert.NoError(t, err)

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

	// Check that the tx was recorded correctly
	txids, err := idx.GetTransactionsIDs(ds, viewKey.(*icrypto.Curve25519PrivateKey))
	assert.NoError(t, err)
	assert.Len(t, txids, 1)

	// Check that we can fetch the proof correctly and that it validates
	proofs, merkleRoot, err := idx.GetTxoProofs(ds, []types.ID{commitment}, [][]byte{ul.Serialize()})
	assert.NoError(t, err)
	assert.Len(t, proofs, 1)

	valid := standard.ValidateInclusionProof(commitment.Bytes(), proofs[0].Index, proofs[0].Hashes, proofs[0].Flags, proofs[0].Accumulator, merkleRoot[:])
	assert.True(t, valid)

	// Close the index, then repo and make sure it loads state correctly
	err = idx.Close(ds)
	assert.NoError(t, err)

	idx, err = NewWalletServerIndex(ds)
	assert.NoError(t, err)

	assert.Len(t, idx.nullifiers, 1)
	assert.Equal(t, idx.bestBlockHeight, uint32(1))
	assert.NotEqual(t, make([]byte, 32), idx.bestBlockID[:])

	// Create a block spending the utxo and make sure the tx is recorded
	nullifier := types.CalculateNullifier(proofs[0].Index, note.Salt, ul.ScriptCommitment, ul.ScriptParams...)
	blk = &blocks.Block{
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

	err = idx.ConnectBlock(dbtx, blk)
	assert.NoError(t, err)
	assert.NoError(t, dbtx.Commit(context.Background()))

	txids, err = idx.GetTransactionsIDs(ds, viewKey.(*icrypto.Curve25519PrivateKey))
	assert.NoError(t, err)
	assert.Len(t, txids, 2)

	// Make sure the indexer deleted the appropriate data after the spend
	proofs, merkleRoot, err = idx.GetTxoProofs(ds, []types.ID{commitment}, [][]byte{ul.Serialize()})
	assert.Error(t, err)
	assert.Len(t, proofs, 0)
	assert.Len(t, idx.nullifiers, 0)

	_, err = dsFetchIndexValue(ds, idx, walletServerNotePrefix+commitment.String())
	assert.Error(t, err)
	_, err = dsFetchIndexValue(ds, idx, walletServerNullifierKeyPrefix+string(privKeyBytes)+"/"+nullifier.String())
	assert.Error(t, err)
}

func randSpendNote() types.SpendNote {
	note := types.SpendNote{
		ScriptHash: make([]byte, 32),
		Amount:     20000,
		AssetID:    types.ID{},
		State:      [128]byte{},
		Salt:       [32]byte{},
	}
	rand.Read(note.ScriptHash)
	rand.Read(note.AssetID[:])
	rand.Read(note.State[:])
	rand.Read(note.Salt[:])
	return note
}
