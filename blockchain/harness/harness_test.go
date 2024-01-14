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
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1), Pregenerate(0))
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
	}

	sighash, err := tx.SigHash()
	assert.NoError(t, err)

	privateParams := standard.PrivateParams{
		Inputs: []standard.PrivateInput{
			{
				SpendNote: types.SpendNote{
					Amount:  notes[0].Note.Amount,
					Salt:    notes[0].Note.Salt,
					AssetID: notes[0].Note.AssetID,
					State:   notes[0].Note.State,
				},
				CommitmentIndex: proof.Index,
				InclusionProof: standard.InclusionProof{
					Hashes: proof.Hashes,
					Flags:  proof.Flags,
				},
				ScriptCommitment: notes[0].LockingScript.ScriptCommitment.Bytes(),
				ScriptParams:     notes[0].LockingScript.LockingParams,
				UnlockingParams:  make([]byte, 64),
			},
		},
		Outputs: []standard.PrivateOutput{
			{
				SpendNote: types.SpendNote{
					ScriptHash: outScriptHash,
					Amount:     outNote.Note.Amount,
					Salt:       outNote.Note.Salt,
					State:      outNote.Note.State,
					AssetID:    outNote.Note.AssetID,
				},
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
	salt, err := types.RandomSalt()
	if err != nil {
		return err
	}

	out := &types.SpendNote{
		ScriptHash: sn.Note.ScriptHash,
		Amount:     sn.Note.Amount / 2,
		AssetID:    sn.Note.AssetID,
		State:      sn.Note.State,
		Salt:       salt,
	}
	commitment, err := out.Commitment()
	if err != nil {
		return err
	}

	salt2, err := types.RandomSalt()
	if err != nil {
		return err
	}

	out2 := &types.SpendNote{
		ScriptHash: sn.Note.ScriptHash,
		Amount:     10000,
		AssetID:    sn.Note.AssetID,
		State:      sn.Note.State,
		Salt:       salt2,
	}
	commitment2, err := out2.Commitment()
	if err != nil {
		return err
	}

	sn2 := &SpendableNote{
		Note:          out,
		LockingScript: sn.LockingScript,
		PrivateKey:    sn.PrivateKey,
	}
	sn3 := &SpendableNote{
		Note:          out2,
		LockingScript: sn.LockingScript,
		PrivateKey:    sn.PrivateKey,
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

	blks, _, err = h.generateBlocks(10000)
	if err != nil {
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

	if err := h.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(tx)}, []*SpendableNote{sn2, sn3}); err != nil {
		return err
	}

	var (
		stakeNullifier types.Nullifier
		sn4            *SpendableNote
	)
	for k, v := range h.spendableNotes {
		if v.Note.Amount == 10000 {
			stakeNullifier = k
			sn4 = v
			break
		}
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

	delete(h.spendableNotes, stakeNullifier)

	f2, err := os.Create("blocks/blocks2.dat")
	if err != nil {
		return err
	}
	defer f.Close()

	blk, err := h.chain.GetBlockByHeight(15001)
	if err != nil {
		return err
	}

	ser, err = proto.Marshal(blk)
	if err != nil {
		return err
	}
	lenBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(ser)))

	if _, err := f2.Write(lenBytes); err != nil {
		return err
	}
	if _, err := f2.Write(ser); err != nil {
		return err
	}

	blk, err = h.chain.GetBlockByHeight(15002)
	if err != nil {
		return err
	}

	ser, err = proto.Marshal(blk)
	if err != nil {
		return err
	}
	lenBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(ser)))

	if _, err := f2.Write(lenBytes); err != nil {
		return err
	}
	if _, err := f2.Write(ser); err != nil {
		return err
	}

	blks, _, err = h.generateBlocks(6000)
	if err != nil {
		return err
	}

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

/*func TestBuildBlockData(t *testing.T) {
	err := generateBlocksDat()
	assert.NoError(t, err)
}*/
