// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package stake_test

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStakeCircuit(t *testing.T) {
	defaultTime := time.Time{}
	defaultTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(defaultTimeBytes, uint64(defaultTime.Unix()))

	scriptCommitment := make([]byte, 32)
	rand.Read(scriptCommitment)

	_, pub1, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	pub1bytes, err := crypto.MarshalPublicKey(pub1)
	assert.NoError(t, err)
	_, pub2, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	pub2bytes, err := crypto.MarshalPublicKey(pub2)
	assert.NoError(t, err)
	_, pub3, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	pub3bytes, err := crypto.MarshalPublicKey(pub3)
	assert.NoError(t, err)

	scriptParams := [][]byte{{0x02}, pub1bytes, pub2bytes, pub3bytes}

	script := types.UnlockingScript{
		ScriptCommitment: scriptCommitment,
		ScriptParams:     scriptParams,
	}
	scriptHash := script.Hash()

	r := make([]byte, 32)
	rand.Read(r)
	var salt [32]byte
	copy(salt[:], r)

	note1 := types.SpendNote{
		ScriptHash: scriptHash[:],
		AssetID:    [32]byte{},
		Amount:     1000000,
		Salt:       salt,
	}

	commitment, err := note1.Commitment()
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

	fakeSig1 := make([]byte, 32)
	rand.Read(fakeSig1)

	fakeSig2 := make([]byte, 32)
	rand.Read(fakeSig2)

	nullifier := types.CalculateNullifier(inclusionProof.Index, note1.Salt, scriptCommitment, scriptParams...)

	privateParams := &stake.PrivateParams{
		AssetID:         note1.AssetID,
		Salt:            note1.Salt,
		State:           [types.StateLen]byte{},
		CommitmentIndex: inclusionProof.Index,
		InclusionProof: standard.InclusionProof{
			Accumulator: inclusionProof.Accumulator,
			Hashes:      inclusionProof.Hashes,
			Flags:       inclusionProof.Flags,
		},
		ScriptCommitment: scriptCommitment,
		ScriptParams:     scriptParams,
		UnlockingParams:  [][]byte{fakeSig1, fakeSig2},
	}

	publicParams := &stake.PublicParams{
		TXORoot:   root.Bytes(),
		SigHash:   sigHash,
		Amount:    uint64(note1.Amount),
		Nullifier: nullifier[:],
	}

	valid := stake.StakeCircuit(privateParams, publicParams)
	assert.True(t, valid)
}
