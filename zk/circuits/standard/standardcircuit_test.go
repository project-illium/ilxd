// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package standard_test

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStandardCircuit(t *testing.T) {
	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	raw, err := pub.Raw()
	assert.NoError(t, err)

	randScriptCommitment := make([]byte, 32)
	rand.Read(randScriptCommitment)

	us := types.UnlockingScript{
		ScriptCommitment: randScriptCommitment,
		ScriptParams:     [][]byte{raw},
	}

	usScriptHash := us.Hash()

	_, pub2, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	raw2, err := pub2.Raw()
	assert.NoError(t, err)

	us2 := types.UnlockingScript{
		ScriptCommitment: randScriptCommitment,
		ScriptParams:     [][]byte{raw2},
	}

	us2ScriptHash := us2.Hash()

	r := make([]byte, 32)
	rand.Read(r)
	var salt [32]byte
	copy(salt[:], r)

	r2 := make([]byte, 32)
	rand.Read(r2)
	var salt2 [32]byte
	copy(salt2[:], r2)

	note1 := types.SpendNote{
		ScriptHash: usScriptHash[:],
		AssetID:    [types.AssetIDLen]byte{},
		Amount:     1000000,
		State:      [types.StateLen]byte{},
		Salt:       salt,
	}

	commitment, err := note1.Commitment()
	assert.NoError(t, err)

	note2 := types.SpendNote{
		ScriptHash: us2ScriptHash[:],
		AssetID:    [types.AssetIDLen]byte{},
		Amount:     990000,
		State:      [types.StateLen]byte{},
		Salt:       salt2,
	}

	outputScriptHash := us2.Hash()

	commitment2, err := note2.Commitment()
	assert.NoError(t, err)

	acc := blockchain.NewAccumulator()
	acc.Insert(commitment[:], true)

	for i := uint32(0); i < 10; i++ {
		iBytes := make([]byte, 32)
		binary.BigEndian.PutUint32(iBytes, i)
		acc.Insert(iBytes, false)
	}

	root := acc.Root()

	inclusionProof, err := acc.GetProof(commitment[:])
	assert.NoError(t, err)

	sigHash := make([]byte, 32)
	rand.Read(sigHash)

	nullifier, err := types.CalculateNullifier(inclusionProof.Index, note1.Salt, us.ScriptCommitment, us.ScriptParams...)
	assert.NoError(t, err)

	fakeSig := make([]byte, 32)
	rand.Read(fakeSig)

	privateParams := &standard.PrivateParams{
		Inputs: []standard.PrivateInput{
			{
				Amount:          uint64(note1.Amount),
				Salt:            note1.Salt,
				AssetID:         [types.AssetIDLen]byte{},
				State:           [types.StateLen]byte{},
				CommitmentIndex: 0,
				InclusionProof: standard.InclusionProof{
					Hashes:      inclusionProof.Hashes,
					Flags:       inclusionProof.Flags,
					Accumulator: inclusionProof.Accumulator,
				},
				ScriptCommitment: us.ScriptCommitment,
				ScriptParams:     us.ScriptParams,
				UnlockingParams:  [][]byte{fakeSig},
			},
		},
		Outputs: []standard.PrivateOutput{
			{
				ScriptHash: outputScriptHash[:],
				Amount:     uint64(note2.Amount),
				Salt:       note2.Salt,
				State:      [types.StateLen]byte{},
				AssetID:    [types.AssetIDLen]byte{},
			},
		},
	}

	publicParams := &standard.PublicParams{
		Outputs: []standard.PublicOutput{
			{
				Commitment: commitment2[:],
			},
		},
		TXORoot:    root[:],
		Coinbase:   0,
		SigHash:    sigHash,
		Fee:        10000,
		Nullifiers: [][]byte{nullifier.Bytes()},
		MintAmount: 0,
		MintID:     nil,
		Locktime:   time.Now(),
	}

	valid := standard.StandardCircuit(privateParams, publicParams)
	assert.True(t, valid)
}
