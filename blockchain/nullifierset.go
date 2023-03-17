// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	datastore "github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"sync"
)

// NullifierSet provides cached access to the nullifier set database.
type NullifierSet struct {
	ds            repo.Datastore
	cachedEntries map[types.Nullifier]bool
	maxEntries    uint
	mtx           sync.RWMutex
}

// NewNullifierSet returns a new NullifierSet. maxEntries controls how
// much memory is used for cache purposes.
func NewNullifierSet(ds repo.Datastore, maxEntries uint) *NullifierSet {
	return &NullifierSet{
		ds:            ds,
		cachedEntries: make(map[types.Nullifier]bool),
		maxEntries:    maxEntries,
		mtx:           sync.RWMutex{},
	}
}

// NullifierExists returns whether or not the nullifier exists in the
// nullifier set. If the entry is cached we'll return from memory, otherwise
// we have to check the disk.
//
// After determining if the nullifier exists we'll update the cache with the
// value. This is useful, for example, if the mempool checks the existence of
// the nullifier, we cache it, then the blockchain doesn't need to hit the disk
// a second time when validating the block.
func (ns *NullifierSet) NullifierExists(nullifier types.Nullifier) (bool, error) {
	ns.mtx.Lock()
	defer ns.mtx.Unlock()

	exists, ok := ns.cachedEntries[nullifier]
	if ok {
		return exists, nil
	}

	exists, err := dsNullifierExists(ns.ds, nullifier)
	if err != nil {
		return false, err
	}

	if ns.maxEntries <= 0 {
		return exists, nil
	}

	ns.limitCache(1)
	ns.cachedEntries[nullifier] = exists
	return exists, nil
}

// AddNullifiers adds the nullifiers to the database using the provided
// database transaction. There is no caching of these writes as the
// nullifiers will never be deleted or mutated so we have to incure the
// write penalty at some point.
func (ns *NullifierSet) AddNullifiers(dbtx datastore.Txn, nullifiers []types.Nullifier) error {
	ns.mtx.Lock()
	defer ns.mtx.Unlock()

	// We're just going to delete the cached entry here rather than
	// update the cache. The reason for this it's unlikely we'll need
	// to check if the nullifier exists again after adding it (this would
	// only happen in a double spend). We also want to avoid having an
	// incorrect value in the cache in case rollback was called on the
	// database transaction.
	for _, n := range nullifiers {
		delete(ns.cachedEntries, n)
	}

	return dsPutNullifiers(dbtx, nullifiers)
}

func (ns *NullifierSet) limitCache(newEntries int) {
	// If adding this new entry will put us over the max number of allowed
	// entries, then evict an entry.
	i := 0
	if uint(len(ns.cachedEntries)+newEntries) > ns.maxEntries {
		// Remove a random entry from the map. Relying on the random
		// starting point of Go's map iteration. It's worth noting that
		// the random iteration starting point is not 100% guaranteed
		// by the spec, however most Go compilers support it.
		// Ultimately, the iteration order isn't important here because
		// in order to manipulate which items are evicted, an adversary
		// would need to be able to execute preimage attacks on the
		// hashing function in order to start eviction at a specific
		// entry.
		for nullifier := range ns.cachedEntries {
			delete(ns.cachedEntries, nullifier)
			i++
			if i >= newEntries {
				break
			}
		}
	}
}
