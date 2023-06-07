// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTestHarness(t *testing.T) {
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1))
	assert.NoError(t, err)

	err = h.GenerateBlocks(1000)
	assert.NoError(t, err)

	notes := h.SpendableNotes()
	inCommitment, err := notes[0].Note.Commitment()

	acc := h.Accumulator()
	proof, err := acc.GetProof(inCommitment)
	assert.NoError(t, err)
	root := acc.Root()

	nullifer, err := types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].UnlockingScript.SnarkVerificationKey, notes[0].UnlockingScript.PublicParams...)
	assert.NoError(t, err)

	var salt [32]byte
	rand.Read(salt[:])
	outUnlockingScript := &types.UnlockingScript{
		SnarkVerificationKey: notes[0].UnlockingScript.SnarkVerificationKey,
		PublicParams:         notes[0].UnlockingScript.PublicParams,
	}
	outScriptHash := outUnlockingScript.Hash()
	outNote := &SpendableNote{
		Note: &types.SpendNote{
			ScriptHash: outScriptHash[:],
			Amount:     notes[0].Note.Amount - 10,
			AssetID:    notes[0].Note.AssetID,
			State:      notes[0].Note.State,
			Salt:       salt,
		},
		UnlockingScript: outUnlockingScript,
		PrivateKey:      notes[0].PrivateKey,
	}

	outCommitment, err := outNote.Note.Commitment()
	assert.NoError(t, err)

	tx := transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: outCommitment,
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{nullifer[:]},
		TxoRoot:    root[:],
		Locktime:   0,
		Fee:        10,
	}

	sighash, err := tx.SigHash()
	assert.NoError(t, err)

	privateParams := standard.PrivateParams{
		Inputs: []standard.PrivateInput{
			{
				Amount:          uint64(notes[0].Note.Amount),
				Salt:            notes[0].Note.Salt,
				AssetID:         notes[0].Note.AssetID,
				State:           notes[0].Note.State,
				CommitmentIndex: proof.Index,
				InclusionProof: standard.InclusionProof{
					Hashes:      proof.Hashes,
					Flags:       proof.Flags,
					Accumulator: proof.Accumulator,
				},
				SnarkVerificationKey: notes[0].UnlockingScript.SnarkVerificationKey,
				UserParams:           notes[0].UnlockingScript.PublicParams,
				SnarkProof:           make([]byte, 100),
			},
		},
		Outputs: []standard.PrivateOutput{
			{
				ScriptHash: outScriptHash[:],
				Amount:     uint64(outNote.Note.Amount),
				Salt:       outNote.Note.Salt,
				State:      outNote.Note.State,
				AssetID:    outNote.Note.AssetID,
			},
		},
	}
	publicParams := standard.PublicParams{
		TXORoot: root[:],
		SigHash: sighash,
		Outputs: []standard.PublicOutput{
			{
				Commitment: tx.Outputs[0].Commitment,
				CipherText: tx.Outputs[0].Ciphertext,
			},
		},
		Nullifiers: tx.Nullifiers,
		Fee:        tx.Fee,
	}
	tx.Proof, err = zk.CreateSnark(standard.StandardCircuit, &privateParams, &publicParams)
	assert.NoError(t, err)

	err = h.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(&tx)}, []*SpendableNote{outNote})
	assert.NoError(t, err)
}
