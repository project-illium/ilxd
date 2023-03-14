// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"github.com/go-test/deep"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPutGetHeader(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutHeader(dbtx, header))
	assert.NoError(t, dbtx.Commit(context.Background()))

	header2, err := dsFetchHeader(ds, header.ID())
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(header, header2))
}

func TestPutGetBlock(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	block := randomBlock(header, 5)
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutBlock(dbtx, block))
	assert.NoError(t, dbtx.Commit(context.Background()))

	exists, err := dsBlockExists(ds, block.ID())
	assert.NoError(t, err)
	assert.True(t, exists)

	block2, err := dsFetchBlock(ds, block.ID())
	assert.NoError(t, err)
	assert.Empty(t, deep.Equal(block, block2))
}

func TestPutGetBlockIDByHeight(t *testing.T) {
	ds := mock.NewMapDatastore()
	header := randomBlockHeader(5, randomID())
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsPutBlockIDFromHeight(dbtx, header.ID(), 5))
	assert.NoError(t, dbtx.Commit(context.Background()))

	id, err := dsFetchBlockIDFromHeight(ds, 5)
	assert.NoError(t, err)
	assert.Equal(t, header.ID(), id)
}
