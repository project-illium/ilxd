// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package standard_test

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStandardCircuit(t *testing.T) {
	_, pub, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	pubx, puby := pub.(*icrypto.NovaPublicKey).ToXY()
	assert.NoError(t, err)

	scriptCommitment := make([]byte, 32)

	us := types.LockingScript{
		ScriptCommitment: types.NewID(scriptCommitment),
		LockingParams:    [][]byte{pubx, puby},
	}

	usScriptHash, err := us.Hash()
	assert.NoError(t, err)

	_, pub2, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	pubx2, puby2 := pub2.(*icrypto.NovaPublicKey).ToXY()
	assert.NoError(t, err)

	us2 := types.LockingScript{
		ScriptCommitment: types.NewID(scriptCommitment),
		LockingParams:    [][]byte{pubx2, puby2},
	}

	us2ScriptHash, err := us2.Hash()
	assert.NoError(t, err)

	salt, err := types.RandomSalt()
	assert.NoError(t, err)

	salt2, err := types.RandomSalt()
	assert.NoError(t, err)

	note1 := types.SpendNote{
		ScriptHash: usScriptHash,
		AssetID:    [types.AssetIDLen]byte{},
		Amount:     1000000,
		State:      types.State{},
		Salt:       salt,
	}

	commitment, err := note1.Commitment()
	assert.NoError(t, err)

	note2 := types.SpendNote{
		ScriptHash: us2ScriptHash,
		AssetID:    [types.AssetIDLen]byte{},
		Amount:     990000,
		State:      types.State{},
		Salt:       salt2,
	}

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

	nullifier, err := types.CalculateNullifier(inclusionProof.Index, note1.Salt, us.ScriptCommitment.Bytes(), us.LockingParams...)
	assert.NoError(t, err)

	fakeSig := make([]byte, 64)
	rand.Read(fakeSig)

	privateParams := &standard.PrivateParams{
		Inputs: []standard.PrivateInput{
			{
				SpendNote: types.SpendNote{
					Amount:  note1.Amount,
					Salt:    note1.Salt,
					AssetID: [types.AssetIDLen]byte{},
					State:   types.State{},
				},
				CommitmentIndex: 0,
				InclusionProof: standard.InclusionProof{
					Hashes: inclusionProof.Hashes,
					Flags:  inclusionProof.Flags,
				},
				ScriptCommitment: us.ScriptCommitment.Bytes(),
				ScriptParams:     us.LockingParams,
				UnlockingParams:  []byte(fmt.Sprintf("(cons 0x%x 0x%x)", fakeSig[:32], fakeSig[32:])),
			},
		},
		Outputs: []standard.PrivateOutput{
			{
				SpendNote: types.SpendNote{
					ScriptHash: us2ScriptHash,
					Amount:     note2.Amount,
					Salt:       note2.Salt,
					State:      types.State{},
					AssetID:    [types.AssetIDLen]byte{},
				},
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
