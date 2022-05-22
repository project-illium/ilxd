// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"crypto/rand"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func mockBlockIndex(ds repo.Datastore, nBlocks int) (*blockIndex, error) {
	err := populateDatabase(ds, nBlocks)
	if err != nil {
		return nil, err
	}
	blockIndex := NewBlockIndex(ds)
	if err := blockIndex.Init(); err != nil {
		return nil, err
	}
	return blockIndex, nil
}

func randomBlockHeader(height uint32, parent types.ID) *blocks.BlockHeader {
	r := make([]byte, 32)
	rand.Read(r)

	header := &blocks.BlockHeader{
		Version: 1,
		Height:  height,
		Parent:  parent[:],
	}
	return header
}

func populateDatabase(ds repo.Datastore, nBlocks int) error {
	dbtx, err := ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}

	prev := randomBlockHeader(0, [32]byte{})
	if err := dsPutHeader(dbtx, prev); err != nil {
		return err
	}

	if err := dsPutBlockIDFromHeight(dbtx, prev.ID(), 0); err != nil {
		return err
	}

	for i := uint32(1); i < uint32(nBlocks); i++ {
		header := randomBlockHeader(i, prev.ID())
		header.Parent = prev.ID().Bytes()

		if err := dsPutHeader(dbtx, header); err != nil {
			return err
		}

		if err := dsPutBlockIDFromHeight(dbtx, header.ID(), i); err != nil {
			return err
		}

		prev = header
	}

	if err := dsPutBlockIndexState(dbtx, &blockNode{
		ds:      ds,
		blockID: prev.ID(),
		height:  5000,
	}); err != nil {
		return err
	}

	return dbtx.Commit(context.Background())
}

func TestBlockIndex(t *testing.T) {
	// Create a new memory datastore and populate it with
	// 5000 block headers.
	ds := mock.NewMapDatastore()
	err := populateDatabase(ds, 5000)
	assert.NoError(t, err)

	// Initialize the index
	index := NewBlockIndex(ds)
	err = index.Init()
	assert.NoError(t, err)
	assert.NotNil(t, index.Tip())

	// Traverse the index backwards from the tip to genesis
	node := index.Tip()
	for i := 0; i < 5000; i++ {
		node, err = node.Parent()
		assert.NoError(t, err)
	}

	// Traverse forward from genesis to tip
	for i := 0; i < 4999; i++ {
		node, err = node.Child()
		assert.NoError(t, err)
	}

	// Test get by height
	header, err := index.GetNodeByHeight(4000)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4000), header.Height())

	// Test get by ID
	header2, err := index.GetNodeByID(header.blockID)
	assert.NoError(t, err)
	assert.Equal(t, header.blockID, header2.blockID)

	// Create new header and extend the index
	newHeader := randomBlockHeader(5001, index.Tip().ID())
	index.ExtendIndex(newHeader)

	assert.Equal(t, newHeader.ID(), index.Tip().ID())

	// Commit the new tip and check the db updated correctly
	txn, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	err = index.Commit(txn)
	assert.NoError(t, err)

	err = txn.Commit(context.Background())
	assert.NoError(t, err)

	node, err = dsFetchBlockIndexState(ds)
	assert.NoError(t, err)
	assert.Equal(t, newHeader.ID(), node.ID())
	assert.Equal(t, newHeader.Height, node.Height())
}
