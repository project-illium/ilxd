// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"crypto/rand"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTxoRootSet(t *testing.T) {
	ds := mock.NewMapDatastore()
	txo := NewTxoRootSet(ds, 5)

	txoRoots := make([]types.ID, 10)
	for i := range txoRoots {
		b := make([]byte, 32)
		rand.Read(b)
		txoRoots[i] = types.NewID(b)
	}

	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	for _, r := range txoRoots {
		err = txo.AddRoot(dbtx, r)
		assert.NoError(t, err)
	}

	err = dbtx.Commit(context.Background())
	assert.NoError(t, err)

	for _, n := range txoRoots {
		exists, err := txo.Exists(n)
		assert.NoError(t, err)
		assert.True(t, exists)
	}

	assert.EqualValues(t, 5, len(txo.cache))

	for i := 0; i < 10; i++ {
		b := make([]byte, 32)
		rand.Read(b)
		exists, err := txo.Exists(types.NewID(b))
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}
