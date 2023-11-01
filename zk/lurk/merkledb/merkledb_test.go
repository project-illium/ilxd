// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package merkledb

import (
	"crypto/rand"
	"crypto/sha256"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerkleDB(t *testing.T) {
	db, err := NewMerkleDB(mock.NewMapDatastore())
	assert.NoError(t, err)

	r := make([]byte, 32)
	rand.Read(r)

	m := make(map[types.ID][]byte)
	for i := 0; i < 100; i++ {
		key := sha256.Sum256(append(r, 0x00, byte(i)))
		value := sha256.Sum256(append(r, 0x01, byte(i)))
		m[types.NewID(key[:])] = value[:]

		err = db.Put(types.NewID(key[:]), value[:])
		assert.NoErrorf(t, err, "%d", i)
	}

	root, err := db.Root()
	assert.NoError(t, err)

	for k, v := range m {
		val, proof, err := db.Get(k)
		assert.NoError(t, err)
		assert.EqualValues(t, v, val)

		valid, err := ValidateInclusionProof(k, v, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)

		exists, proof, err := db.Exists(k)
		assert.NoError(t, err)
		assert.True(t, exists)

		valid, err = ValidateInclusionProof(k, v, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	}

	for k := range m {
		err := db.Delete(k)
		assert.NoError(t, err)

		_, _, err = db.Get(k)
		assert.Error(t, err)

		exists, proof, err := db.Exists(k)
		assert.NoError(t, err)
		assert.False(t, exists)

		root, err := db.Root()
		assert.NoError(t, err)

		valid, err := ValidateInclusionProof(k, nil, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	}
}
