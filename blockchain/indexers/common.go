// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	datastore "github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types/blocks"
)

// Indexer defines the interface that all blockchain indexers will use.
type Indexer interface {
	// Key returns the key of the index as a string.
	Key() string

	// Name returns the human-readable name of the index.
	Name() string

	// ConnectBlock is called when a block is connected to the chain.
	// The indexer can use this opportunity to parse it and store it in
	// the database. The database transaction must be respected.
	//
	// It is the responsibility of the index to update the height
	// of the index in the datastore. This can be done after connecting
	// each block or, if the index is doing some caching, on Close().
	ConnectBlock(dbtx datastore.Txn, blk *blocks.Block) error

	// Close is called when the index manager shuts down and gives the indexer
	// an opportunity to do some cleanup.
	Close(ds repo.Datastore) error
}
