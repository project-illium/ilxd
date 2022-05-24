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

type NullifierSet struct {
	ds            repo.Datastore
	cachedEntries map[types.Nullifier]bool
	maxEntries    uint
	mtx           sync.RWMutex
}

func NewNullifierSet(ds repo.Datastore, maxEntries uint) *NullifierSet {
	return &NullifierSet{
		ds:            ds,
		cachedEntries: make(map[types.Nullifier]bool),
		maxEntries:    maxEntries,
		mtx:           sync.RWMutex{},
	}
}

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

func (ns *NullifierSet) AddNullifiers(dbtx datastore.Txn, nullifiers []types.Nullifier) error {
	ns.mtx.Lock()
	defer ns.mtx.Unlock()

	if err := dsPutNullifiers(dbtx, nullifiers); err != nil {
		return err
	}

	if ns.maxEntries <= 0 {
		return nil
	}

	ns.limitCache(len(nullifiers))
	for _, n := range nullifiers {
		ns.cachedEntries[n] = true
	}
	return nil
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
