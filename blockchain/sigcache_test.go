// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSigCache(t *testing.T) {
	max := 10
	cache := NewSigCache(uint(max))

	for i := 0; i < 11; i++ {
		sig := make([]byte, 64)
		rand.Read(sig)
		_, pk, err := icrypto.GenerateNovaKey(rand.Reader)
		assert.NoError(t, err)

		sigHash := hash.HashFunc(sig)
		cache.Add(types.NewID(sigHash), sig, pk)
		assert.True(t, cache.Exists(types.NewID(sigHash), sig, pk))
		assert.LessOrEqual(t, len(cache.validSigs), max)
	}
}
