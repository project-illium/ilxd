// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/blockchain/utils"
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

	h1 := utils.HashWithIndex(d1, 0)
	h2 := utils.HashWithIndex(d2, 1)
	h3 := utils.HashWithIndex(d3, 2)
	h4 := utils.HashWithIndex(d4, 3)
	h5 := utils.HashWithIndex(d5, 4)
	h6 := utils.HashWithIndex(d6, 5)
	h7 := utils.HashWithIndex(d7, 6)
	h8 := utils.HashWithIndex(d8, 7)

	h12 := utils.HashMerkleBranches(h1, h2)
	h34 := utils.HashMerkleBranches(h3, h4)
	h56 := utils.HashMerkleBranches(h5, h6)
	h78 := utils.HashMerkleBranches(h7, h8)

	h1234 := utils.HashMerkleBranches(h12, h34)
	h5678 := utils.HashMerkleBranches(h56, h78)

	h12345678 := utils.HashMerkleBranches(h1234, h5678)

	acc := NewAccumulator()

	acc.Insert(d1, false)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h1})), acc.Root())

	acc.Insert(d2, false)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h12})), acc.Root())

	acc.Insert(d3, false)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h3, h12})), acc.Root())

	acc.Insert(d4, true)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h1234})), acc.Root())

	acc.Insert(d5, true)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h5, h1234})), acc.Root())

	acc.Insert(d6, false)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h56, h1234})), acc.Root())

	acc.Insert(d7, true)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h7, h56, h1234})), acc.Root())

	acc.Insert(d8, false)
	assert.Equal(t, models.NewID(utils.CatAndHash([][]byte{h12345678})), acc.Root())

	proof4, err := acc.GetProof(d4)
	assert.NoError(t, err)
	assert.Equal(t, acc.Root(), models.NewID(utils.CatAndHash(proof4.Accumulator)))
	assert.Equal(t, models.NewID(d4), proof4.ID)
	assert.Equal(t, [][]byte{h3, h12, h5678}, proof4.Hashes)
	assert.Equal(t, uint64(4), proof4.Flags)
	assert.Equal(t, uint64(3), proof4.Index)

	proof5, err := acc.GetProof(d5)
	assert.NoError(t, err)
	assert.Equal(t, acc.Root(), models.NewID(utils.CatAndHash(proof5.Accumulator)))
	assert.Equal(t, models.NewID(d5), proof5.ID)
	assert.Equal(t, [][]byte{h6, h78, h1234}, proof5.Hashes)
	assert.Equal(t, uint64(3), proof5.Flags)
	assert.Equal(t, uint64(4), proof5.Index)

	proof7, err := acc.GetProof(d7)
	assert.NoError(t, err)
	assert.Equal(t, acc.Root(), models.NewID(utils.CatAndHash(proof7.Accumulator)))
	assert.Equal(t, models.NewID(d7), proof7.ID)
	assert.Equal(t, [][]byte{h8, h56, h1234}, proof7.Hashes)
	assert.Equal(t, uint64(1), proof7.Flags)
	assert.Equal(t, uint64(6), proof7.Index)

	acc.DropProof(d7)
	_, err = acc.GetProof(d7)
	assert.Error(t, err)
}
