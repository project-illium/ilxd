// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockstore

// Blockstore is an interface to a custom datastore for storing blocks.
// This is separated from the main database interface to enable optimized
// writes.
type Blockstore interface {
	// PutBlockData puts the block data, usually serialized transactions,
	// to the datastore.
	PutBlockData(height uint32, blockData []byte) (BlockLocation, error)

	// FetchBlockData returns the block data from the datastore
	FetchBlockData(location BlockLocation) ([]byte, error)

	// DeleteBefore deletes all block data before the provided height
	DeleteBefore(height uint32) error
}
