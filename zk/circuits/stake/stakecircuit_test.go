// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package stake_test

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
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

	_, pub1, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)
	_, pub2, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)
	_, pub3, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	pub1x, pub1y := pub1.(*icrypto.NovaPublicKey).ToXY()
	pub2x, pub2y := pub2.(*icrypto.NovaPublicKey).ToXY()
	pub3x, pub3y := pub3.(*icrypto.NovaPublicKey).ToXY()
	scriptParams := [][]byte{{0x02}, pub1x, pub1y, pub2x, pub2y, pub3x, pub3y}

	script := types.UnlockingScript{
		ScriptCommitment: scriptCommitment,
		ScriptParams:     scriptParams,
	}
	scriptHash, err := script.Hash()
	assert.NoError(t, err)

	salt, err := types.RandomSalt()
	assert.NoError(t, err)

	note1 := types.SpendNote{
		ScriptHash: scriptHash[:],
		AssetID:    [32]byte{},
		Amount:     1000000,
		Salt:       salt,
	}

	commitment := note1.Commitment()

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

	nullifier, err := types.CalculateNullifier(inclusionProof.Index, note1.Salt, scriptCommitment, scriptParams...)
	assert.NoError(t, err)

	privateParams := &stake.PrivateParams{
		AssetID:         note1.AssetID,
		Salt:            note1.Salt,
		State:           [types.StateLen]byte{},
		CommitmentIndex: inclusionProof.Index,
		InclusionProof: standard.InclusionProof{
			Hashes: inclusionProof.Hashes,
			Flags:  inclusionProof.Flags,
		},
		ScriptCommitment: scriptCommitment,
		ScriptParams:     scriptParams,
		UnlockingParams:  []byte(fmt.Sprintf("(cons 0x%x 0x%x)", fakeSig1, fakeSig2)),
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
