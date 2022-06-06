// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package stake_test

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/wallet"
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
		Pubkeys: []*wallet.TimeLockedPubkey{
			{PubKey: pub1},
			{PubKey: pub2},
			{PubKey: pub3},
		},
	}
	pubkeys := make([][]byte, 3)
	for _, key := range ss.Pubkeys {
		b, err := key.Serialize()
		assert.NoError(t, err)
		pubkeys = append(pubkeys, b)
	}

	r := make([]byte, 32)
	rand.Read(r)
	var salt [32]byte
	copy(salt[:], r)

	note1 := wallet.SpendNote{
		SpendScript: ss,
		AssetID:     [32]byte{},
		Amount:      1000000,
		Salt:        salt,
	}

	commitment, err := note1.Commitment()
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

	nullifier, err := types.CalculateNullifier(inclusionProof.Index, note1.Salt, ss.Threshold, pubkeys...)
	assert.NoError(t, err)

	privateParams := &stake.PrivateParams{
		AssetID:         note1.AssetID[:],
		Salt:            note1.Salt[:],
		CommitmentIndex: inclusionProof.Index,
		InclusionProof: standard.InclusionProof{
			Accumulator: inclusionProof.Accumulator,
			Hashes:      inclusionProof.Hashes,
			Flags:       inclusionProof.Flags,
		},
		Threshold:   ss.Threshold,
		Pubkeys:     [][]byte{append(raw1, defaultTimeBytes...), append(raw2, defaultTimeBytes...), append(raw3, defaultTimeBytes...)},
		Signatures:  [][]byte{sig2, sig3},
		SigBitfield: 6,
	}

	publicParams := &stake.PublicParams{
		TXORoot:   root.Bytes(),
		SigHash:   sigHash,
		Amount:    note1.Amount,
		Nullifier: nullifier[:],
		Blocktime: time.Now(),
	}

	valid := stake.StakeCircuit(privateParams, publicParams)
	assert.True(t, valid)
}
