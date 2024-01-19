// Copyright (c) 2024 The illium developers
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

func TestNullifierSet(t *testing.T) {
	ds := mock.NewMapDatastore()
	ns := NewNullifierSet(ds, 5)

	nullifiers := make([]types.Nullifier, 10)
	for i := range nullifiers {
		b := make([]byte, 32)
		rand.Read(b)
		nullifiers[i] = types.NewNullifier(b)
	}

	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	err = ns.AddNullifiers(dbtx, nullifiers)
	assert.NoError(t, err)

	err = dbtx.Commit(context.Background())
	assert.NoError(t, err)

	for _, n := range nullifiers {
		exists, err := ns.NullifierExists(n)
		assert.NoError(t, err)
		assert.True(t, exists)
	}

	assert.EqualValues(t, 5, len(ns.cachedEntries))

	for i := 0; i < 10; i++ {
		b := make([]byte, 32)
		rand.Read(b)
		exists, err := ns.NullifierExists(types.NewNullifier(b))
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}
