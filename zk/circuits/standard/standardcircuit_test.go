// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package standard

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/models"
	"github.com/project-illium/ilxd/wallet"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStandardCircuit(t *testing.T) {
	defaultTime := time.Time{}
	defaultTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(defaultTimeBytes, uint64(defaultTime.Unix()))

	_, pub1, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	priv2, pub2, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	priv3, pub3, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	raw1, err := pub1.Raw()
	assert.NoError(t, err)
	raw2, err := pub2.Raw()
	assert.NoError(t, err)
	raw3, err := pub3.Raw()
	assert.NoError(t, err)

	ss := wallet.SpendScript{
		Threshold: 2,
		Pubkeys: []wallet.TimedPubkey{
			{Pubkey: pub1},
			{Pubkey: pub2},
			{Pubkey: pub3},
		},
	}
	pubkeys := make([][]byte, 3)
	for _, key := range ss.Pubkeys {
		b, err := key.Serialize()
		assert.NoError(t, err)
		pubkeys = append(pubkeys, b)
	}

	_, pub4, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	ss2 := wallet.SpendScript{
		Threshold: 1,
		Pubkeys:   []wallet.TimedPubkey{{Pubkey: pub4}},
	}

	outputSpendScript, err := ss2.Hash()
	assert.NoError(t, err)

	r := make([]byte, 32)
	rand.Read(r)
	var salt [32]byte
	copy(salt[:], r)

	r2 := make([]byte, 32)
	rand.Read(r2)
	var salt2 [32]byte
	copy(salt2[:], r2)

	ocp := wallet.OutputCommitmentPreimage{
		SpendScript: ss,
		AssetID:     [32]byte{},
		Amount:      1000000,
		Salt:        salt,
	}

	commitment, err := ocp.Commitment()
	assert.NoError(t, err)

	ocp2 := wallet.OutputCommitmentPreimage{
		SpendScript: ss2,
		AssetID:     [32]byte{},
		Amount:      990000,
		Salt:        salt2,
	}

	commitment2, err := ocp2.Commitment()
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

	sig2, err := priv2.Sign(sigHash)
	assert.NoError(t, err)

	sig3, err := priv3.Sign(sigHash)
	assert.NoError(t, err)

	nullifier, err := models.CalculateNullifier(inclusionProof.Index, ocp.Salt, ss.Threshold, pubkeys...)
	assert.NoError(t, err)

	privateParams := PrivateParams{
		Inputs: []PrivateInput{
			{
				Signatures:      [][]byte{sig2, sig3},
				SigBitfield:     6,
				Salt:            ocp.Salt[:],
				Amount:          ocp.Amount,
				AssetID:         ocp.AssetID,
				Threshold:       ss.Threshold,
				Pubkeys:         [][]byte{append(raw1, defaultTimeBytes...), append(raw2, defaultTimeBytes...), append(raw3, defaultTimeBytes...)},
				CommitmentIndex: inclusionProof.Index,
				InclusionProof: InclusionProof{
					Accumulator: inclusionProof.Accumulator,
					Hashes:      inclusionProof.Hashes,
					Flags:       inclusionProof.Flags,
				},
			},
		},
		Outputs: []PrivateOutput{
			{
				AssetID:     [32]byte{},
				Amount:      990000,
				Salt:        salt2[:],
				SpendScript: outputSpendScript,
			},
		},
	}

	publicParams := PublicParams{
		TXORoot:           root[:],
		OutputCommitments: [][]byte{commitment2},
		Coinbase:          0,
		SigHash:           sigHash,
		Fee:               10000,
		Nullifiers:        [][32]byte{nullifier},
		MintAmount:        0,
		MintID:            nil,
		Blocktime:         time.Now(),
	}

	valid := StandardCircuit(privateParams, publicParams)
	assert.True(t, valid)
}
