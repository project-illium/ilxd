// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/types/blocks"
)

// GetBlockFunc defines a function that is part of the blockchain
// class that gets us around circular imports.
type GetBlockFunc func(height uint32) (*blocks.Block, error)

// Indexer defines the interface that all blockchain indexers will use.
type Indexer interface {
	// Key returns the key of the index as a string.
	Key() string

	// Name returns the human-readable name of the index.
	Name() string

	// ConnectBlock is called when a block is connected to the chain.
	// The indexer can use this opportunity to parse it and store it in
	// the database. The database transaction must be respected.
	ConnectBlock(dbtx datastore.Txn, blk *blocks.Block) error
}
