// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProofValidator(t *testing.T) {
	proofCache := NewProofCache(10)
	proofValidator := NewProofValidator(proofCache)

	salt1, err := types.RandomSalt()
	assert.NoError(t, err)

	spendKey, _, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	mockScriptCommitment := make([]byte, 32)

	pubx, puby := spendKey.GetPublic().(*icrypto.NovaPublicKey).ToXY()

	inUnlockingScript := types.UnlockingScript{
		ScriptCommitment: mockScriptCommitment,
		ScriptParams:     [][]byte{pubx, puby},
	}
	inScriptHash, err := inUnlockingScript.Hash()
	assert.NoError(t, err)
	inNote := &types.SpendNote{
		ScriptHash: inScriptHash[:],
		Amount:     1000000,
		AssetID:    types.IlliumCoinID,
		Salt:       salt1,
		State:      [types.StateLen]byte{},
	}
	inCommitment := inNote.Commitment()

	outUnlockingScript := types.UnlockingScript{
		ScriptCommitment: mockScriptCommitment,
		ScriptParams:     [][]byte{pubx, puby},
	}
	outScriptHash, err := outUnlockingScript.Hash()
	assert.NoError(t, err)
	outNote := &types.SpendNote{
		ScriptHash: outScriptHash[:],
		Amount:     900000,
		AssetID:    types.IlliumCoinID,
		Salt:       salt1,
		State:      [types.StateLen]byte{},
	}
	outCommitment := outNote.Commitment()

	acc := NewAccumulator()
	acc.Insert(inCommitment[:], true)
	root := acc.Root()

	inNullifier, err := types.CalculateNullifier(0, inNote.Salt, inUnlockingScript.ScriptCommitment, inUnlockingScript.ScriptParams...)
	assert.NoError(t, err)

	fakeProof := make([]byte, 8000)
	rand.Read(fakeProof)

	standardTx := &transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: outCommitment[:],
			},
		},
		Nullifiers: [][]byte{inNullifier[:]},
		TxoRoot:    root[:],
		Locktime:   nil,
		Fee:        100000,
		Proof:      fakeProof,
	}
	stakeTx := &transactions.StakeTransaction{
		Amount:    1000000,
		Nullifier: inNullifier[:],
		TxoRoot:   root[:],
		Proof:     fakeProof,
	}
	mintID := make([]byte, 32)
	rand.Read(mintID)
	mintTx := &transactions.MintTransaction{
		Type:      transactions.MintTransaction_FIXED_SUPPLY,
		Asset_ID:  mintID,
		NewTokens: 100,
		Outputs: []*transactions.Output{
			{
				Commitment: outCommitment[:],
			},
		},
		Fee:        0,
		Nullifiers: [][]byte{inNullifier[:]},
		TxoRoot:    root[:],
		Proof:      fakeProof,
	}
	treasuryTx := &transactions.TreasuryTransaction{
		Amount: 2000000,
		Outputs: []*transactions.Output{
			{
				Commitment: outCommitment[:],
			},
		},
		Proof: fakeProof,
	}
	coinbaseTx := &transactions.CoinbaseTransaction{
		NewCoins: 3000000,
		Outputs: []*transactions.Output{
			{
				Commitment: outCommitment[:],
			},
		},
		Proof: fakeProof,
	}

	err = proofValidator.Validate([]*transactions.Transaction{
		transactions.WrapTransaction(standardTx),
		transactions.WrapTransaction(stakeTx),
		transactions.WrapTransaction(mintTx),
		transactions.WrapTransaction(treasuryTx),
		transactions.WrapTransaction(coinbaseTx),
	})
	assert.NoError(t, err)

	c := ValidateTransactionProof(transactions.WrapTransaction(coinbaseTx), NewProofCache(10))
	err = <-c
	assert.NoError(t, err)
}
