// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package hash

import (
	"golang.org/x/crypto/blake2s"
)

const HashSize = 32

func HashFunc(data []byte) []byte {
	h := blake2s.Sum256(data)
	return h[:]
}
