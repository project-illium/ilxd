// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/repo/datastore"
	"github.com/project-illium/ilxd/types/blocks"
)

type IndexManager interface {
	Init(tipHeight uint32, getBlock func(height uint32) (*blocks.Block, error)) error
	ConnectBlock(dbtx datastore.Txn, blk *blocks.Block) error
	Close() error
}
