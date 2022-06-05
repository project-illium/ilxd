// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"sync"
)

type TxoRootSet struct {
	cache      map[types.ID]bool
	mtx        sync.RWMutex
	ds         repo.Datastore
	maxEntries uint
}

func NewTxoRootSet(ds repo.Datastore, maxEntires uint) *TxoRootSet {
	return &TxoRootSet{
		cache:      make(map[types.ID]bool),
		mtx:        sync.RWMutex{},
		ds:         ds,
		maxEntries: maxEntires,
	}
}

func (t *TxoRootSet) Exists(txoRoot types.ID) (bool, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	if exists := t.cache[txoRoot]; exists {
		return true, nil
	}

	return dsTxoSetRootExists(t.ds, txoRoot)
}

func (t *TxoRootSet) AddRoot(dbtx datastore.Txn, txoRoot types.ID) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	err := dsPutTxoSetRoot(dbtx, txoRoot)
	if err != nil {
		return err
	}

	if t.maxEntries <= 0 {
		return nil
	}

	// If adding this new entry will put us over the max number of allowed
	// entries, then evict an entry.
	if uint(len(t.cache)+1) > t.maxEntries {
		// Remove a random entry from the map. Relying on the random
		// starting point of Go's map iteration. It's worth noting that
		// the random iteration starting point is not 100% guaranteed
		// by the spec, however most Go compilers support it.
		// Ultimately, the iteration order isn't important here because
		// in order to manipulate which items are evicted, an adversary
		// would need to be able to execute preimage attacks on the
		// hashing function in order to start eviction at a specific
		// entry.
		for entry := range t.cache {
			delete(t.cache, entry)
			break
		}
	}
	t.cache[txoRoot] = true
	return nil
}
