// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package hash

import (
	"golang.org/x/crypto/blake2s"
)

const HashSize = 32

// HashFunc is a wrapper around the blake2s hash function.
func HashFunc(data []byte) []byte {
	h := blake2s.Sum256(data)
	return h[:]
}

// Blake2slurk is a modified version of the blake2s hash
// function which sets the first two bits of the output to
// zero. This is needed for use inside a lurk program which
// has a max field element value less than the maximum value
// of the blake2s hash.
func Blake2slurk(data []byte) []byte {
	h := blake2s.Sum256(data)
	h[0] &= 0b00111111
	return h[:]
}
