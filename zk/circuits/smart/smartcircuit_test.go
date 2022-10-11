// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package smart_test

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/circuits/smart"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSmartCircuit(t *testing.T) {
	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	raw, err := pub.Raw()
	assert.NoError(t, err)

	randVerificationKey := make([]byte, 32)
	rand.Read(randVerificationKey)

	us := types.UnlockingScript{
		SnarkVerificationKey: randVerificationKey,
		PublicParams:         [][]byte{raw},
	}

	_, pub2, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	raw2, err := pub2.Raw()
	assert.NoError(t, err)

	us2 := types.UnlockingScript{
		SnarkVerificationKey: randVerificationKey,
		PublicParams:         [][]byte{raw2},
	}

	r := make([]byte, 32)
	rand.Read(r)
	var salt [32]byte
	copy(salt[:], r)

	r2 := make([]byte, 32)
	rand.Read(r2)
	var salt2 [32]byte
	copy(salt2[:], r2)

	note1 := types.SpendNote{
		UnlockingScript: us,
		AssetID:         [32]byte{},
		Amount:          1000000,
		State:           [32]byte{},
		Salt:            salt,
	}

	commitment, err := note1.Commitment()
	assert.NoError(t, err)

	note2 := types.SpendNote{
		UnlockingScript: us2,
		AssetID:         [32]byte{},
		Amount:          990000,
		State:           [32]byte{},
		Salt:            salt2,
	}

	outputScriptHash := us2.Hash()

	commitment2, err := note2.Commitment()
	assert.NoError(t, err)

	acc := blockchain.NewAccumulator()
	acc.Insert(commitment, true)

	for i := uint32(0); i < 10; i++ {
		iBytes := make([]byte, 32)
		binary.BigEndian.PutUint32(iBytes, i)
		acc.Insert(iBytes, false)
	}

	root := acc.Root()

	inclusionProof, err := acc.GetProof(commitment)
	assert.NoError(t, err)

	sigHash := make([]byte, 32)
	rand.Read(sigHash)

	nullifier, err := types.CalculateNullifier(inclusionProof.Index, note1.Salt, us.SnarkVerificationKey, us.PublicParams...)
	assert.NoError(t, err)

	fakeSnarkProof := make([]byte, 32)
	rand.Read(fakeSnarkProof)

	privateParams := &smart.PrivateParams{
		Inputs: []smart.PrivateInput{
			{
				Amount:          note1.Amount,
				Salt:            note1.Salt,
				AssetID:         [32]byte{},
				State:           [32]byte{},
				CommitmentIndex: 0,
				InclusionProof: smart.InclusionProof{
					Hashes:      inclusionProof.Hashes,
					Flags:       inclusionProof.Flags,
					Accumulator: inclusionProof.Accumulator,
				},
				SnarkVerificationKey: note1.UnlockingScript.SnarkVerificationKey,
				UserParams:           note1.UnlockingScript.PublicParams,
				SnarkProof:           fakeSnarkProof,
			},
		},
		Outputs: []smart.PrivateOutput{
			{
				ScriptHash: outputScriptHash[:],
				Amount:     note2.Amount,
				Salt:       note2.Salt,
				State:      [32]byte{},
				AssetID:    [32]byte{},
			},
		},
	}

	publicParams := &smart.PublicParams{
		Outputs: []smart.PublicOutput{
			{
				Commitment: commitment2,
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

	valid := smart.SmartCircuit(privateParams, publicParams)
	assert.True(t, valid)
}
