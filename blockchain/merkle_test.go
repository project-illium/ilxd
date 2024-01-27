// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerkleTreeStore(t *testing.T) {
	d1 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 1})
	d2 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2})
	d3 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3})
	d4 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4})
	d5 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 5})
	d6 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 6})
	d7 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7})
	d8 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 8})

	h1 := d1.ID()
	h2 := d2.ID()
	h3 := d3.ID()
	h4 := d4.ID()
	h5 := d5.ID()
	h6 := d6.ID()
	h7 := d7.ID()
	h8 := d8.ID()

	h12 := hash.HashMerkleBranches(h1[:], h2[:])
	h34 := hash.HashMerkleBranches(h3[:], h4[:])
	h56 := hash.HashMerkleBranches(h5[:], h6[:])
	h78 := hash.HashMerkleBranches(h7[:], h8[:])

	h1234 := hash.HashMerkleBranches(h12, h34)
	h5678 := hash.HashMerkleBranches(h56, h78)

	h12345678 := hash.HashMerkleBranches(h1234, h5678)

	merkles := BuildMerkleTreeStore([]types.ID{h1, h2, h3, h4, h5, h6, h7, h8})

	assert.EqualValues(t, h12345678, merkles[len(merkles)-1])
}

func TestMerkleInclusionProof(t *testing.T) {
	d1 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 1})
	d2 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2})
	d3 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3})
	d4 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4})
	d5 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 5})
	d6 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 6})
	d7 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7})
	d8 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 8})

	h1 := d1.ID()
	h2 := d2.ID()
	h3 := d3.ID()
	h4 := d4.ID()
	h5 := d5.ID()
	h6 := d6.ID()
	h7 := d7.ID()
	h8 := d8.ID()

	h12 := hash.HashMerkleBranches(h1[:], h2[:])
	h34 := hash.HashMerkleBranches(h3[:], h4[:])
	h56 := hash.HashMerkleBranches(h5[:], h6[:])
	h78 := hash.HashMerkleBranches(h7[:], h8[:])

	h1234 := hash.HashMerkleBranches(h12, h34)
	h5678 := hash.HashMerkleBranches(h56, h78)

	merkles := BuildMerkleTreeStore([]types.ID{h1, h2, h3, h4, h5, h6, h7, h8})

	hashes, flags := MerkleInclusionProof(merkles, h3)
	assert.Len(t, hashes, 3)
	assert.Equal(t, hashes[0], h4[:])
	assert.Equal(t, hashes[1], h12[:])
	assert.Equal(t, hashes[2], h5678[:])
	assert.Equal(t, uint32(5), flags)

	hashes, flags = MerkleInclusionProof(merkles, h8)
	assert.Len(t, hashes, 3)
	assert.Equal(t, hashes[0], h7[:])
	assert.Equal(t, hashes[1], h56[:])
	assert.Equal(t, hashes[2], h1234[:])
	assert.Equal(t, uint32(0), flags)
}

func TestTransactionsMerkleRoot(t *testing.T) {
	d1 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 1, Proof: []byte{0x01}})
	d2 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2, Proof: []byte{0x02}})
	d3 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3, Proof: []byte{0x03}})
	d4 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4, Proof: []byte{0x04}})
	d5 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 5, Proof: []byte{0x05}})
	d6 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 6, Proof: []byte{0x06}})
	d7 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7, Proof: []byte{0x07}})
	d8 := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 8, Proof: []byte{0x08}})

	id1 := d1.ID()
	id2 := d2.ID()
	id3 := d3.ID()
	id4 := d4.ID()
	id5 := d5.ID()
	id6 := d6.ID()
	id7 := d7.ID()
	id8 := d8.ID()

	u12 := hash.HashMerkleBranches(id1[:], id2[:])
	u34 := hash.HashMerkleBranches(id3[:], id4[:])
	u56 := hash.HashMerkleBranches(id5[:], id6[:])
	u78 := hash.HashMerkleBranches(id7[:], id8[:])

	u1234 := hash.HashMerkleBranches(u12, u34)
	u5678 := hash.HashMerkleBranches(u56, u78)

	root := hash.HashMerkleBranches(u1234, u5678)

	merkleRoot := TransactionsMerkleRoot([]*transactions.Transaction{d1, d2, d3, d4, d5, d6, d7, d8})
	assert.EqualValues(t, root, merkleRoot[:])
}
