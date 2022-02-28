// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/models"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAccumulator(t *testing.T) {
	d1 := []byte{0x01}
	d2 := []byte{0x02}
	d3 := []byte{0x03}
	d4 := []byte{0x04}
	d5 := []byte{0x05}
	d6 := []byte{0x06}
	d7 := []byte{0x07}
	d8 := []byte{0x08}

	h1 := hashWithIndex(d1, 0)
	h2 := hashWithIndex(d2, 1)
	h3 := hashWithIndex(d3, 2)
	h4 := hashWithIndex(d4, 3)
	h5 := hashWithIndex(d5, 4)
	h6 := hashWithIndex(d6, 5)
	h7 := hashWithIndex(d7, 6)
	h8 := hashWithIndex(d8, 7)

	h12 := hashMerkleBranches(h1, h2)
	h34 := hashMerkleBranches(h3, h4)
	h56 := hashMerkleBranches(h5, h6)
	h78 := hashMerkleBranches(h7, h8)

	h1234 := hashMerkleBranches(h12, h34)
	h5678 := hashMerkleBranches(h56, h78)

	h12345678 := hashMerkleBranches(h1234, h5678)

	acc := NewAccumulator()

	acc.Insert(d1, false)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h1})), acc.Root())

	acc.Insert(d2, false)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h12})), acc.Root())

	acc.Insert(d3, false)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h3, h12})), acc.Root())

	acc.Insert(d4, true)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h1234})), acc.Root())

	acc.Insert(d5, true)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h5, h1234})), acc.Root())

	acc.Insert(d6, false)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h56, h1234})), acc.Root())

	acc.Insert(d7, true)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h7, h56, h1234})), acc.Root())

	acc.Insert(d8, false)
	assert.Equal(t, models.NewID(catAndHash([][]byte{h12345678})), acc.Root())

	proof4, err := acc.GetProof(d4)
	assert.NoError(t, err)
	assert.Equal(t, acc.Root(), models.NewID(catAndHash(proof4.Accumulator)))
	assert.Equal(t, models.NewID(d4), proof4.ID)
	assert.Equal(t, [][]byte{h3, h12, h5678}, proof4.Hashes)
	assert.Equal(t, uint64(4), proof4.Flags)

	proof5, err := acc.GetProof(d5)
	assert.NoError(t, err)
	assert.Equal(t, acc.Root(), models.NewID(catAndHash(proof5.Accumulator)))
	assert.Equal(t, models.NewID(d5), proof5.ID)
	assert.Equal(t, [][]byte{h6, h78, h1234}, proof5.Hashes)
	assert.Equal(t, uint64(3), proof5.Flags)

	proof7, err := acc.GetProof(d7)
	assert.NoError(t, err)
	assert.Equal(t, acc.Root(), models.NewID(catAndHash(proof7.Accumulator)))
	assert.Equal(t, models.NewID(d7), proof7.ID)
	assert.Equal(t, [][]byte{h8, h56, h1234}, proof7.Hashes)
	assert.Equal(t, uint64(1), proof7.Flags)

	acc.DropProof(d7)
	_, err = acc.GetProof(d7)
	assert.Error(t, err)
}
