// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"fmt"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"sync"
)

type nullifierEntry struct {
	exists bool
	dirty  bool
}

type NullifierSet struct {
	ds            repo.Datastore
	cachedEntries map[types.Nullifier]*nullifierEntry
	maxCacheSize  int
	totalMemory   int
	mtx           sync.RWMutex
}

func NewNullifierSet(ds repo.Datastore, maxCacheSize int) *NullifierSet {
	return &NullifierSet{
		ds:            ds,
		cachedEntries: make(map[types.Nullifier]*nullifierEntry),
		maxCacheSize:  maxCacheSize,
		mtx:           sync.RWMutex{},
	}
}

func (ns *NullifierSet) Init(tip *blockNode) error {
	consistencyStatus, err := dsFetchNullifierSetConsistencyStatus(ns.ds)
	if err != nil {
		return err
	}
	lastFlushHeight, err := dsFetchNullifierLastFlushHeight(ns.ds)
	if err != nil {
		return err
	}

	switch consistencyStatus {
	case scsConsistent:
		if lastFlushHeight == tip.Height() {
			// Nothing to do.
			// We're good.
			return nil
		} else if lastFlushHeight < tip.Height() {
			// Load the missing blocks from disk and
			// apply any changes to the nullifier set.
			var (
				node = tip
				err  error
			)
			for {
				node, err = tip.Parent()
				if err != nil {
					return err
				}
				if node.height == lastFlushHeight+1 {
					break
				}
			}
			for {
				blk, err := node.Block()
				if err != nil {
					return err
				}
				if err := ns.AddNullifiers(blk.Nullifiers(), blk.Header.Height, flushIfNeeded); err != nil {
					return err
				}
				if node.height == tip.height {
					break
				}
				node, err = node.Child()
				if err != nil {
					return err
				}
			}
			if err := ns.Flush(flushRequired, tip.height); err != nil {
				return err
			}
		} else if lastFlushHeight > tip.Height() {
			// This really should never happen.
			// If we're here it's unlikely the tip node
			// has any attached children that we can use
			// to load the blocks and remove the validator
			// changes from the set. Panic?
		}
	case scsFlushOngoing:
		// Iterate over all the blocks after lastFlushHeight
		// and remove any changes that may have been applied.
		// Traverse the blocks forward from lastFlushHeight to
		// the tip and apply the changes.
		var (
			node = tip
			err  error
		)
		dbtx, err := ns.ds.NewTransaction(context.Background(), false)
		if err != nil {
			return err
		}
		defer dbtx.Discard(context.Background())

		node, err = tip.Parent()
		if err != nil {
			return err
		}
		for {
			blk, err := node.Block()
			if err != nil {
				return err
			}
			if err := dsDeleteNullifiers(dbtx, blk.Nullifiers()); err != nil {
				return err
			}
			if node.height == lastFlushHeight+1 {
				break
			}
			node, err = node.Parent()
			if err != nil {
				return err
			}
		}
		for {
			blk, err := node.Block()
			if err != nil {
				return err
			}
			if err := ns.AddNullifiers(blk.Nullifiers(), blk.Header.Height, flushIfNeeded); err != nil {
				return err
			}
			if node.height == tip.height {
				break
			}
			node, err = node.Child()
			if err != nil {
				return err
			}
		}
		if err := ns.Flush(flushRequired, tip.height); err != nil {
			return err
		}
	case scsEmpty:
		// New node. Just set the consistency state.
		if err := ns.Flush(flushRequired, tip.height); err != nil {
			return err
		}

	}
	return nil
}

func (ns *NullifierSet) NullifierExists(nullifier types.Nullifier) (bool, error) {
	ns.mtx.Lock()
	defer ns.mtx.Unlock()

	entry, ok := ns.cachedEntries[nullifier]
	if ok {
		return entry.exists, nil
	}

	exists, err := dsNullifierExists(ns.ds, nullifier)
	if err != nil {
		return false, err
	}
	ns.totalMemory += 33
	if exists {
		ns.cachedEntries[nullifier] = &nullifierEntry{
			exists: true,
			dirty:  false,
		}
		return true, nil
	} else {
		ns.cachedEntries[nullifier] = &nullifierEntry{
			exists: false,
			dirty:  false,
		}
		return false, nil
	}
}

func (ns *NullifierSet) AddNullifiers(nullifiers []types.Nullifier, chainHeight uint32, flushMode flushMode) error {
	ns.mtx.Lock()
	defer ns.mtx.Unlock()

	for _, n := range nullifiers {
		ns.cachedEntries[n] = &nullifierEntry{
			exists: true,
			dirty:  true,
		}
		ns.totalMemory += 33
	}

	return ns.flush(flushMode, chainHeight)
}

func (ns *NullifierSet) Flush(mode flushMode, chainHeight uint32) error {
	ns.mtx.Lock()
	ns.mtx.Unlock()

	return ns.flush(mode, chainHeight)
}

func (ns *NullifierSet) flush(mode flushMode, chainHeight uint32) error {
	switch mode {
	case flushRequired:
		if err := ns.flushToDisk(chainHeight); err != nil {
			return err
		}
		ns.cachedEntries = make(map[types.Nullifier]*nullifierEntry)
		ns.totalMemory = 0
		return nil
	case flushIfNeeded:
		if ns.totalMemory > ns.maxCacheSize {
			if err := ns.flushToDisk(chainHeight); err != nil {
				return err
			}
			ns.cachedEntries = make(map[types.Nullifier]*nullifierEntry)
			ns.totalMemory = 0
		}
		return nil
	default:
		return fmt.Errorf("unsupported flushmode for the nullifier set")
	}
}

func (ns *NullifierSet) flushToDisk(chainHeight uint32) error {
	if len(ns.cachedEntries) == 0 {
		return nil
	}
	nullifiers := make([]types.Nullifier, 0, len(ns.cachedEntries))
	for n, entry := range ns.cachedEntries {
		if entry.dirty {
			nullifiers = append(nullifiers, n)
		}
	}
	if err := dsPutNullifierSetConsistencyStatus(ns.ds, scsFlushOngoing); err != nil {
		return err
	}

	dbtx, err := ns.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())

	if err := dsPutNullifiers(dbtx, nullifiers); err != nil {
		return err
	}
	if err := dsPutNullifierLastFlushHeight(dbtx, chainHeight); err != nil {
		return err
	}
	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}
	if err := dsPutNullifierSetConsistencyStatus(ns.ds, scsConsistent); err != nil {
		return err
	}

	return nil
}
