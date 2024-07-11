// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package merkledb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerkleDB(t *testing.T) {
	db, err := NewMerkleDB(mock.NewMockDatastore())
	assert.NoError(t, err)

	r := make([]byte, 32)
	rand.Read(r)

	m := make(map[types.ID][]byte)
	for i := 0; i < 1000; i++ {
		iBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(iBytes, uint32(i))
		key := sha256.Sum256(append(r, append([]byte{0x00}, iBytes...)...))
		value := sha256.Sum256(append(r, append([]byte{0x01}, iBytes...)...))

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

		valid, err := ValidateProof(k, v, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)

		exists, proof, err := db.Exists(k)
		assert.NoError(t, err)
		assert.True(t, exists)

		valid, err = ValidateProof(k, v, root, proof)
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

		valid, err := ValidateProof(k, nil, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	}
}

func TestPutOverride(t *testing.T) {
	db, err := NewMerkleDB(mock.NewMockDatastore())
	assert.NoError(t, err)

	r := make([]byte, 32)
	rand.Read(r)

	m := make(map[types.ID][]byte)
	for i := 0; i < 10; i++ {
		iBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(iBytes, uint32(i))
		key := sha256.Sum256(append(r, append([]byte{0x00}, iBytes...)...))
		value := sha256.Sum256(append(r, append([]byte{0x01}, iBytes...)...))

		m[types.NewID(key[:])] = value[:]

		err = db.Put(types.NewID(key[:]), value[:])
		assert.NoErrorf(t, err, "%d", i)
	}

	r = make([]byte, 32)
	rand.Read(r)
	i := 0
	for k := range m {
		iBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(iBytes, uint32(i))
		value := sha256.Sum256(append(r, append([]byte{0x01}, iBytes...)...))

		m[types.NewID(k[:])] = value[:]

		err = db.Put(types.NewID(k[:]), value[:])
		assert.NoErrorf(t, err, "%d", i)
		i++
	}

	root, err := db.Root()
	assert.NoError(t, err)

	for k, v := range m {
		val, proof, err := db.Get(k)
		assert.NoError(t, err)
		assert.EqualValues(t, v, val)

		valid, err := ValidateProof(k, v, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)

		exists, proof, err := db.Exists(k)
		assert.NoError(t, err)
		assert.True(t, exists)

		valid, err = ValidateProof(k, v, root, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	}
}

func TestRootHashAfterDelete(t *testing.T) {
	db, err := NewMerkleDB(mock.NewMockDatastore())
	assert.NoError(t, err)

	r := make([]byte, 32)
	rand.Read(r)

	m := make(map[types.ID][]byte)
	for i := 0; i < 10; i++ {
		iBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(iBytes, uint32(i))
		key := sha256.Sum256(append(r, append([]byte{0x00}, iBytes...)...))
		value := sha256.Sum256(append(r, append([]byte{0x01}, iBytes...)...))

		m[types.NewID(key[:])] = value[:]

		err = db.Put(types.NewID(key[:]), value[:])
		assert.NoErrorf(t, err, "%d", i)
	}

	rand.Read(r)
	for i := 0; i < 50; i++ {
		iBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(iBytes, uint32(i))
		key := sha256.Sum256(append(r, append([]byte{0x00}, iBytes...)...))
		value := sha256.Sum256(append(r, append([]byte{0x01}, iBytes...)...))

		m[types.NewID(key[:])] = value[:]

		root, err := db.Root()
		assert.NoError(t, err)

		err = db.Put(types.NewID(key[:]), value[:])
		assert.NoErrorf(t, err, "%d", i)

		err = db.Delete(types.NewID(key[:]))
		assert.NoError(t, err)

		root2, err := db.Root()
		assert.NoError(t, err)

		assert.Equal(t, root, root2)
	}

}
