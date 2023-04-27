// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import "github.com/project-illium/ilxd/net"

type SyncManager struct {
	network      *net.Network
	chainService *ChainService
}

func NewSyncManager() *SyncManager {
	return &SyncManager{}
}

// - pick peer and sync headers to first checkpoint in memory
// - if last header doesn't match checkpoint ban and start over
// - if header matches backfill blocks.
// - if any block doesn't match hash, ban peer and select a new sync peer.
// - pick a different sync peer and repeat up to the next checkpoint.
//
// - after last checkpoint query peers for best height and hash.
// - select earliest height, if necessary query other peers to
//   confirm they have that block.
// - if yes, sync to that height.
// - if no,
