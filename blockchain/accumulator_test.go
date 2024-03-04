// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
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

	h1 := hash.HashWithIndex(d1, 0)
	h2 := hash.HashWithIndex(d2, 1)
	h3 := hash.HashWithIndex(d3, 2)
	h4 := hash.HashWithIndex(d4, 3)
	h5 := hash.HashWithIndex(d5, 4)
	h6 := hash.HashWithIndex(d6, 5)
	h7 := hash.HashWithIndex(d7, 6)
	h8 := hash.HashWithIndex(d8, 7)

	h12 := hash.HashMerkleBranches(h1, h2)
	h34 := hash.HashMerkleBranches(h3, h4)
	h56 := hash.HashMerkleBranches(h5, h6)
	h78 := hash.HashMerkleBranches(h7, h8)

	h1234 := hash.HashMerkleBranches(h12, h34)
	h5678 := hash.HashMerkleBranches(h56, h78)

	h12345678 := hash.HashMerkleBranches(h1234, h5678)

	acc := NewAccumulator()

	acc.Insert(d1, true)
	assert.Equal(t, types.NewID(h1), acc.Root())

	acc.Insert(d2, true)
	assert.Equal(t, types.NewID(h12), acc.Root())

	acc.Insert(d3, true)
	assert.Equal(t, types.NewID(hash.HashMerkleBranches(h12, h3)), acc.Root())

	acc.Insert(d4, true)
	assert.Equal(t, types.NewID(h1234), acc.Root())

	acc.Insert(d5, true)
	assert.Equal(t, types.NewID(hash.HashMerkleBranches(h1234, h5)), acc.Root())

	acc.Insert(d6, true)
	assert.Equal(t, types.NewID(hash.HashMerkleBranches(h1234, h56)), acc.Root())

	acc.Insert(d7, true)
	h123456 := hash.HashMerkleBranches(h1234, h56)
	h77 := hash.HashMerkleBranches(h7, h7)
	assert.Equal(t, types.NewID(hash.HashMerkleBranches(h123456, h77)), acc.Root())

	acc.Insert(d8, true)
	assert.Equal(t, types.NewID(h12345678), acc.Root())

	proof4, err := acc.GetProof(d4)
	assert.NoError(t, err)
	assert.Equal(t, types.NewID(d4), proof4.ID)
	assert.Equal(t, [][]byte{h3, h12, h5678}, proof4.Hashes)
	assert.Equal(t, uint64(4), proof4.Flags)
	assert.Equal(t, uint64(3), proof4.Index)

	proof5, err := acc.GetProof(d5)
	assert.NoError(t, err)
	assert.Equal(t, types.NewID(d5), proof5.ID)
	assert.Equal(t, [][]byte{h6, h78, h1234}, proof5.Hashes)
	assert.Equal(t, uint64(3), proof5.Flags)
	assert.Equal(t, uint64(4), proof5.Index)

	proof7, err := acc.GetProof(d7)
	assert.NoError(t, err)
	assert.Equal(t, types.NewID(d7), proof7.ID)
	assert.Equal(t, [][]byte{h8, h56, h1234}, proof7.Hashes)
	assert.Equal(t, uint64(1), proof7.Flags)
	assert.Equal(t, uint64(6), proof7.Index)

	acc.DropProof(d7)
	_, err = acc.GetProof(d7)
	assert.Error(t, err)
}

func TestAccumulator_GetProof(t *testing.T) {
	a := NewAccumulator()
	n := 128
	elements := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 32)
		rand.Read(b)
		elements = append(elements, b)
		a.Insert(b, true)

		// Test proof validity as accumulator is growing.
		for x, c := range elements {
			proof, err := a.GetProof(c)
			assert.NoError(t, err)
			valid, err := ValidateInclusionProof(proof.ID.Bytes(), proof.Index, proof.Hashes, proof.Flags, a.Root().Bytes())
			assert.NoError(t, err)
			assert.True(t, valid)
			if !valid {
				fmt.Println(i, x)
				break
			}
		}
	}
}

func TestAccumulator_MergeProofs(t *testing.T) {
	a := NewAccumulator()
	b := NewAccumulator()
	n := 128
	elements := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		d := make([]byte, 32)
		rand.Read(d)
		elements = append(elements, d)

		a.Insert(d, i <= 63)
		b.Insert(d, i > 63)
	}
	a.MergeProofs(b)
	assert.Len(t, a.proofs, 128)
	assert.Len(t, a.lookupMap, 128)

	for _, c := range elements {
		proof, err := a.GetProof(c)
		assert.NoError(t, err)
		valid, err := ValidateInclusionProof(proof.ID.Bytes(), proof.Index, proof.Hashes, proof.Flags, a.Root().Bytes())
		assert.NoError(t, err)
		assert.True(t, valid)
	}
}

func TestAccumulator_Clone(t *testing.T) {
	acc := NewAccumulator()

	assert.True(t, accumulatorDeepEqual(acc, acc.Clone()))

	for i := 0; i < 1000; i++ {
		b := make([]byte, 32)
		rand.Read(b)
		acc.Insert(b, b[0] < 128)

		// Test clone is correct as accumulator is growing.
		assert.True(t, accumulatorDeepEqual(acc, acc.Clone()))
	}
}

func accumulatorDeepEqual(a, b *Accumulator) bool {
	if len(a.acc) != len(b.acc) {
		return false
	}
	for i := range a.acc {
		if !bytes.Equal(a.acc[i], b.acc[i]) {
			return false
		}
	}
	if a.nElements != b.nElements {
		return false
	}
	if !deepEqualProofMap(a.proofs, b.proofs) {
		return false
	}
	for k, v := range a.lookupMap {
		v2, ok := b.lookupMap[k]
		if !ok || v2 != v {
			return false
		}
	}
	return true
}

func deepEqualProofMap(a, b map[types.ID]*InclusionProof) bool {
	if len(a) != len(b) {
		return false
	}
	for ak, av := range a {
		bv := b[ak]
		if av.ID != bv.ID {
			return false
		}
		if len(av.Hashes) != len(bv.Hashes) {
			return false
		}
		for i := range av.Hashes {
			if !bytes.Equal(av.Hashes[i], bv.Hashes[i]) {
				return false
			}
		}
		if av.Flags != bv.Flags {
			return false
		}
		if av.Index != bv.Index {
			return false
		}
		if !bytes.Equal(av.last, bv.last) {
			return false
		}
	}
	return true
}

func TestDeserializeAccumulator(t *testing.T) {
	commitment, err := hex.DecodeString("2bc0f879e43eb5bdafe89af6a96cb2402dce6da2f6ca3603d2500fc39d03fc4a")
	assert.NoError(t, err)

	index := uint64(10000)

	root, err := hex.DecodeString("0fa432c84e24bac3179bd0452413957e949e3762ff1338d7926455311e369193")
	assert.NoError(t, err)

	flags := uint64(2)

	h0, err := hex.DecodeString("04df052f49d035e9d95cb42f4a4d4b2301f3e4dc8f6ec8092672f7c19c5bc4ff")
	assert.NoError(t, err)
	h1, err := hex.DecodeString("1ce43d9334bec09ede47906007df920bdb819c546abbc27dbbc87eec9d62b74f")
	assert.NoError(t, err)
	h2, err := hex.DecodeString("1b33760ceea9c7b463699d5cc15cfdd1376757118b9b39bf663aba367f42badb")
	assert.NoError(t, err)

	valid, err := ValidateInclusionProof(commitment, index, [][]byte{h0, h1, h2}, flags, root)
	assert.NoError(t, err)
	assert.True(t, valid)
}
