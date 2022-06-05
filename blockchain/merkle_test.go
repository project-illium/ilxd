// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/params/hash"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerkleTreeStore(t *testing.T) {
	d1 := []byte{0x01}
	d2 := []byte{0x02}
	d3 := []byte{0x03}
	d4 := []byte{0x04}
	d5 := []byte{0x05}
	d6 := []byte{0x06}
	d7 := []byte{0x07}
	d8 := []byte{0x08}

	h1 := hash.HashFunc(d1)
	h2 := hash.HashFunc(d2)
	h3 := hash.HashFunc(d3)
	h4 := hash.HashFunc(d4)
	h5 := hash.HashFunc(d5)
	h6 := hash.HashFunc(d6)
	h7 := hash.HashFunc(d7)
	h8 := hash.HashFunc(d8)

	h12 := hash.HashMerkleBranches(h1, h2)
	h34 := hash.HashMerkleBranches(h3, h4)
	h56 := hash.HashMerkleBranches(h5, h6)
	h78 := hash.HashMerkleBranches(h7, h8)

	h1234 := hash.HashMerkleBranches(h12, h34)
	h5678 := hash.HashMerkleBranches(h56, h78)

	h12345678 := hash.HashMerkleBranches(h1234, h5678)

	merkles := BuildMerkleTreeStore([][]byte{d1, d2, d3, d4, d5, d6, d7, d8})

	assert.EqualValues(t, h12345678, merkles[len(merkles)-1])
}
