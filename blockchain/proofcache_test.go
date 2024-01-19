// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProofCache(t *testing.T) {
	max := 10
	cache := NewProofCache(uint(max))

	for i := 0; i < 11; i++ {
		proof := make([]byte, 64)
		rand.Read(proof)

		txid := randomID()

		proofHash := hash.HashFunc(proof)
		cache.Add(types.NewID(proofHash), proof, txid)
		assert.True(t, cache.Exists(types.NewID(proofHash), proof, txid))
		assert.LessOrEqual(t, len(cache.validProofs), max)
	}
}
