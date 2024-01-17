// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNewTestHarness(t *testing.T) {
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1))
	assert.NoError(t, err)

	err = h.GenerateBlocks(5)
	assert.NoError(t, err)

	notes := h.SpendableNotes()
	inCommitment, err := notes[0].Note.Commitment()
	assert.NoError(t, err)

	acc := h.Accumulator()
	proof, err := acc.GetProof(inCommitment[:])
	assert.NoError(t, err)
	root := acc.Root()

	nullifer, err := types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].LockingScript.ScriptCommitment.Bytes(), notes[0].LockingScript.LockingParams...)
	assert.NoError(t, err)

	salt, err := types.RandomSalt()
	assert.NoError(t, err)

	outLockingScript := &types.LockingScript{
		ScriptCommitment: notes[0].LockingScript.ScriptCommitment,
		LockingParams:    notes[0].LockingScript.LockingParams,
	}
	outScriptHash, err := outLockingScript.Hash()
	assert.NoError(t, err)

	outNote := &SpendableNote{
		Note: &types.SpendNote{
			ScriptHash: outScriptHash,
			Amount:     notes[0].Note.Amount - 10,
			AssetID:    notes[0].Note.AssetID,
			State:      notes[0].Note.State,
			Salt:       salt,
		},
		LockingScript: outLockingScript,
		PrivateKey:    notes[0].PrivateKey,
	}

	outCommitment, err := outNote.Note.Commitment()
	assert.NoError(t, err)

	tx := transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: outCommitment[:],
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{nullifer[:]},
		TxoRoot:    root[:],
		Fee:        10,
		Proof:      make([]byte, 11000),
	}

	err = h.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(&tx)}, []*SpendableNote{outNote})
	assert.NoError(t, err)
}

func TestTestHarness_Accumulator(t *testing.T) {
	f, err := os.Create("blocks/blocks.dat")
	assert.NoError(t, err)
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), WriteToFile(f))
	assert.NoError(t, err)

	err = h.GenerateBlocks(21000)
	assert.NoError(t, err)

	h.Close()

	f2, err := os.Open("blocks/blocks.dat")
	assert.NoError(t, err)

	f3, err := os.Create("blocks/blocks2.dat")
	assert.NoError(t, err)

	h2, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), LoadBlocks(f2, 15000), WriteToFile(f3))
	assert.NoError(t, err)

	err = h2.GenerateBlocks(6000)
	assert.NoError(t, err)

	h2.Close()
}
