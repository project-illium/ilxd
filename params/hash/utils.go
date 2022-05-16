// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package hash

import "encoding/binary"

// HashMerkleBranches takes two hashes, treated as the left and right tree
// nodes, and returns the hash of their concatenation.  This is a helper
// function used to aid in the generation of a merkle tree.
func HashMerkleBranches(left []byte, right []byte) []byte {
	// Concatenate the left and right nodes.
	var h [HashSize * 2]byte
	copy(h[:HashSize], left[:])
	copy(h[HashSize:], right[:])

	return HashFunc(h[:])
}

// HashWithIndex prepends the index to data before hashing.
func HashWithIndex(data []byte, index uint64) []byte {
	d := make([]byte, len(data)+8)
	copy(d[:8], nElementsToBytes(index))
	copy(d[8:], data)
	return HashFunc(d)
}

// CatAndHash concatenates all the elements in the slice together
// and then hashes.
func CatAndHash(data [][]byte) []byte {
	combined := make([]byte, 0, HashSize*len(data))
	for _, peak := range data {
		peakCopy := make([]byte, len(peak))
		copy(peakCopy, peak)
		combined = append(combined, peakCopy...)
	}
	return HashFunc(combined)
}

// nElementsToBytes converts a uint64 to bytes.
func nElementsToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}
