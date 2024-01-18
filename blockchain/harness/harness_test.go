// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
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

/*func TestTestHarness_GenerateBlocksDat(t *testing.T) {
	b1, err := os.Open("blocks/blocks.dat")
	assert.NoError(t, err)

	f2, err := os.Create("blocks/blocks2.dat")
	assert.NoError(t, err)

	h2, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), LoadBlocks(b1, 15000), WriteToFile(f2))
	assert.NoError(t, err)

	err = h2.GenerateBlocks(1)
	assert.NoError(t, err)

	sk, pk, err := lcrypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	pid, err := peer.IDFromPublicKey(pk)
	assert.NoError(t, err)
	pidBytes, err := pid.Marshal()
	assert.NoError(t, err)

	notes := h2.SpendableNotes()
	commitment, err := notes[0].Note.Commitment()
	assert.NoError(t, err)
	proof, err := h2.Accumulator().GetProof(commitment.Bytes())
	assert.NoError(t, err)
	nullifer, err := types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].LockingScript.ScriptCommitment.Bytes(), notes[0].LockingScript.LockingParams...)
	assert.NoError(t, err)

	sh, err := notes[0].LockingScript.Hash()
	assert.NoError(t, err)

	salt0, err := types.RandomSalt()
	assert.NoError(t, err)

	salt1, err := types.RandomSalt()
	assert.NoError(t, err)

	note0 := types.SpendNote{
		ScriptHash: sh,
		Amount:     notes[0].Note.Amount / 2,
		AssetID:    types.IlliumCoinID,
		Salt:       salt0,
		State:      nil,
	}

	commit0, err := note0.Commitment()
	assert.NoError(t, err)

	note1 := types.SpendNote{
		ScriptHash: sh,
		Amount:     notes[0].Note.Amount / 2,
		AssetID:    types.IlliumCoinID,
		Salt:       salt1,
		State:      nil,
	}

	commit1, err := note1.Commitment()
	assert.NoError(t, err)

	stdTx := &transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: commit0.Bytes(),
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
			{
				Commitment: commit1.Bytes(),
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{nullifer.Bytes()},
		TxoRoot:    h2.Accumulator().Root().Bytes(),
		Locktime:   nil,
		Fee:        1000,
		Proof:      []byte{0x00},
	}

	createdNotes := []*SpendableNote{
		{
			Note:             &note0,
			LockingScript:    notes[0].LockingScript,
			PrivateKey:       notes[0].PrivateKey,
			cachedScriptHash: sh,
		},
		{
			Note:             &note1,
			LockingScript:    notes[0].LockingScript,
			PrivateKey:       notes[0].PrivateKey,
			cachedScriptHash: sh,
		},
	}

	err = h2.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(stdTx)}, createdNotes)
	assert.NoError(t, err)

	notes = h2.SpendableNotes()
	commitment, err = notes[0].Note.Commitment()
	assert.NoError(t, err)
	proof, err = h2.Accumulator().GetProof(commitment.Bytes())
	assert.NoError(t, err)
	nullifer, err = types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].LockingScript.ScriptCommitment.Bytes(), notes[0].LockingScript.LockingParams...)
	assert.NoError(t, err)

	stakeTx := &transactions.StakeTransaction{
		Validator_ID: pidBytes,
		Amount:       uint64(notes[0].Note.Amount),
		Nullifier:    nullifer.Bytes(),
		TxoRoot:      h2.Accumulator().Root().Bytes(),
		LockedUntil:  0,
		Signature:    nil,
		Proof:        []byte{0x00},
	}
	sigHash, err := stakeTx.SigHash()
	assert.NoError(t, err)
	sig, err := sk.Sign(sigHash)
	assert.NoError(t, err)
	stakeTx.Signature = sig

	err = h2.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(stakeTx)}, nil)
	assert.NoError(t, err)

	err = h2.GenerateBlocks(6000)
	assert.NoError(t, err)

	h2.Close()

}

func TestTestHarness_Blockchain(t *testing.T) {
	f, err := BlocksData.Open("blocks/blocks.dat")
	assert.NoError(t, err)

	h2, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), LoadBlocks(f, 6))
	assert.NoError(t, err)
	err = h2.GenerateBlocks(1)
	assert.NoError(t, err)
}*/
