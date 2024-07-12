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

	// FetchBlockRegion returns the given region inside a block from disk
	FetchBlockRegion(location BlockLocation, offset, numBytes uint32) ([]byte, error)

	// DeleteBefore will delete all block files containing heights before
	// the provided height, but will not delete the file containing the
	// provided height unless the height is the last block in the file.
	DeleteBefore(height uint32) error
}
