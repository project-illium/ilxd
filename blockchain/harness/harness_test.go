// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"os"
	"testing"
)

func TestNewTestHarness(t *testing.T) {
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), Pregenerate(15000))
	assert.NoError(t, err)

	err = h.GenerateBlocks(5)
	assert.NoError(t, err)

	notes := h.SpendableNotes()
	inCommitment := notes[0].Note.Commitment()

	acc := h.Accumulator()
	proof, err := acc.GetProof(inCommitment[:])
	assert.NoError(t, err)
	root := acc.Root()

	nullifer := types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].UnlockingScript.ScriptCommitment, notes[0].UnlockingScript.ScriptParams...)

	var salt [32]byte
	rand.Read(salt[:])
	outUnlockingScript := &types.UnlockingScript{
		ScriptCommitment: notes[0].UnlockingScript.ScriptCommitment,
		ScriptParams:     notes[0].UnlockingScript.ScriptParams,
	}
	outScriptHash, err := outUnlockingScript.Hash()
	assert.NoError(t, err)

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

	outCommitment := outNote.Note.Commitment()

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
					Hashes: proof.Hashes,
					Flags:  proof.Flags,
				},
				ScriptCommitment: notes[0].UnlockingScript.ScriptCommitment,
				ScriptParams:     notes[0].UnlockingScript.ScriptParams,
				UnlockingParams:  make([]byte, 64),
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

func generateBlocksDat() error {
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), Pregenerate(0))
	if err != nil {
		return err
	}

	f, err := os.Create("blocks/blocks.dat")
	if err != nil {
		return err
	}
	defer f.Close()

	blks, spendableNotes, err := h.generateBlocks(15000)
	if err != nil {
		return err
	}

	genesis, err := h.chain.GetBlockByHeight(0)
	if err != nil {
		return err
	}

	ser, err := proto.Marshal(genesis)
	if err != nil {
		return err
	}
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(ser)))

	if _, err := f.Write(lenBytes); err != nil {
		return err
	}
	if _, err := f.Write(ser); err != nil {
		return err
	}

	for _, blk := range blks {
		ser, err := proto.Marshal(blk)
		if err != nil {
			return err
		}
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(len(ser)))

		if _, err := f.Write(lenBytes); err != nil {
			return err
		}
		if _, err := f.Write(ser); err != nil {
			return err
		}
	}

	for _, blk := range blks {
		if err := h.chain.ConnectBlock(blk, blockchain.BFFastAdd); err != nil {
			return err
		}
		for _, out := range blk.Outputs() {
			h.acc.Insert(out.Commitment, true)
		}
	}
	h.spendableNotes = spendableNotes

	var (
		nullifier types.Nullifier
		sn        *SpendableNote
	)
	for k, v := range h.spendableNotes {
		nullifier = k
		sn = v
		break
	}

	out := &types.SpendNote{
		ScriptHash: sn.Note.ScriptHash,
		Amount:     sn.Note.Amount / 2,
		AssetID:    sn.Note.AssetID,
		State:      sn.Note.State,
	}
	rand.Read(out.Salt[:])
	commitment := out.Commitment()

	out2 := &types.SpendNote{
		ScriptHash: sn.Note.ScriptHash,
		Amount:     sn.Note.Amount / 2,
		AssetID:    sn.Note.AssetID,
		State:      sn.Note.State,
	}
	rand.Read(out2.Salt[:])
	commitment2 := out2.Commitment()

	sn2 := &SpendableNote{
		Note:            out,
		UnlockingScript: sn.UnlockingScript,
		PrivateKey:      sn.PrivateKey,
	}
	sn3 := &SpendableNote{
		Note:            out2,
		UnlockingScript: sn.UnlockingScript,
		PrivateKey:      sn.PrivateKey,
	}

	tx := &transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: commitment[:],
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
			{
				Commitment: commitment2[:],
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{nullifier.Bytes()},
		TxoRoot:    h.acc.Root().Bytes(),
	}
	if err := h.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(tx)}, []*SpendableNote{sn2, sn3}); err != nil {
		return err
	}

	var (
		stakeNullifier types.Nullifier
		sn4            *SpendableNote
	)
	for k, v := range h.spendableNotes {
		stakeNullifier = k
		sn4 = v
		break
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return err
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return err
	}
	pidBytes, err := pid.Marshal()
	if err != nil {
		return err
	}

	stx := &transactions.StakeTransaction{
		Validator_ID: pidBytes,
		Amount:       uint64(sn4.Note.Amount),
		Nullifier:    stakeNullifier[:],
		TxoRoot:      h.acc.Root().Bytes(),
	}

	if err := h.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(stx)}, []*SpendableNote{}); err != nil {
		return err
	}

	blks, _, err = h.generateBlocks(9998)
	if err != nil {
		return err
	}

	f2, err := os.Create("blocks/blocks2.dat")
	if err != nil {
		return err
	}
	defer f.Close()

	for _, blk := range blks {
		ser, err := proto.Marshal(blk)
		if err != nil {
			return err
		}
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(len(ser)))

		if _, err := f2.Write(lenBytes); err != nil {
			return err
		}
		if _, err := f2.Write(ser); err != nil {
			return err
		}
	}

	return nil
}
