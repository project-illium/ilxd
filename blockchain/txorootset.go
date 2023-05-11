// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"sync"
	"time"
)

// TxoRootSet holds the full set of txoRoots for the entire chain. A new
// root is created with each block so this set grows with each block.
//
// In the future we could consider limiting the set to N blocks from the tip
// to reduce storage, however this would make transactions that were signed
// prior to N but not broadcasted invalid. For now we'll just store everything.
type TxoRootSet struct {
	cache      map[types.ID]time.Time
	mtx        sync.RWMutex
	ds         repo.Datastore
	maxEntries uint
}

// NewTxoRootSet returns a new TxoRootSet. maxEntries can be used to control the
// amount of memory used by the cache.
func NewTxoRootSet(ds repo.Datastore, maxEntires uint) *TxoRootSet {
	return &TxoRootSet{
		cache:      make(map[types.ID]time.Time),
		mtx:        sync.RWMutex{},
		ds:         ds,
		maxEntries: maxEntires,
	}
}

// RootExists checks whether the root exists in the set. The memory cache holds
// the most recent roots as those are most likely to be used in transactions. If
// the root is in memory we can skip going to disk.
func (t *TxoRootSet) RootExists(txoRoot types.ID) (bool, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	if _, exists := t.cache[txoRoot]; exists {
		return true, nil
	}

	return dsTxoSetRootExists(t.ds, txoRoot)
}

// AddRoot will add a new root to the set.
//
// NOTE: This adds the root to disk storage only as it's expected this method will
// be called as part of a broader database transaction. We don't update the memory
// cache here because we don't want it to get out of sync with the disk if there
// is a rollback on the database transaction.
//
// Ideally database interface would support commit hooks for this purpose but
// that will require wrapping the libp2p interface and building out a new db
// implementation. For now we will expect the caller to use the UpdateCache
// method upon a successful database transaction commit.
func (t *TxoRootSet) AddRoot(dbtx datastore.Txn, txoRoot types.ID) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return dsPutTxoSetRoot(dbtx, txoRoot)
}

// UpdateCache will add the new txoRoot to the memory cache. If the new entry
// would cause the cache to exceed maxEntires, the oldest entry will be evicted.
//
// NOTE: This should be called after successfully committing the root via the
// AddRoot function. It's no big deal if this functionality is not used as
// this will just trigger root lookups from disk, however no root should ever
// be added to the cache that isn't also on disk.
func (t *TxoRootSet) UpdateCache(txoRoot types.ID) {
	if t.maxEntries <= 0 {
		return
	}

	// If adding this new entry will put us over the max number of allowed
	// entries, then evict an entry.
	if uint(len(t.cache)+1) > t.maxEntries {
		var (
			toEvict  types.ID
			earliest = time.Now()
		)
		for entry, birthday := range t.cache {
			if birthday.Before(earliest) {
				earliest = birthday
				toEvict = entry
			}
		}
		delete(t.cache, toEvict)
	}
	t.cache[txoRoot] = time.Now()
}

// Clone returns a copy of the TxoRootSet
func (t *TxoRootSet) Clone() *TxoRootSet {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	ret := &TxoRootSet{
		cache:      make(map[types.ID]time.Time),
		mtx:        sync.RWMutex{},
		ds:         t.ds,
		maxEntries: t.maxEntries,
	}
	for txoRoot, timestamp := range t.cache {
		ret.cache[txoRoot] = timestamp
	}
	return ret
}
