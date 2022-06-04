// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"fmt"
	"github.com/project-illium/ilxd/repo"
	"sync"
	"time"
)

type AccumulatorDB struct {
	acc       *Accumulator
	ds        repo.Datastore
	lastFlush time.Time
	mtx       sync.RWMutex
}

func NewAccumulatorDB(ds repo.Datastore) *AccumulatorDB {
	return &AccumulatorDB{
		acc: NewAccumulator(),
		ds:  ds,
		mtx: sync.RWMutex{},
	}
}

func (adb *AccumulatorDB) Init(tip *blockNode) error {
	consistencyStatus, err := dsFetchAccumulatorSetConsistencyStatus(adb.ds)
	if err != nil {
		return err
	}
	lastFlushHeight, err := dsFetchAccumulatorLastFlushHeight(adb.ds)
	if err != nil {
		return err
	}

	switch consistencyStatus {
	case scsConsistent:
		acc, err := dsFetchAccumulator(adb.ds)
		if err != nil {
			return err
		}
		adb.acc = acc
		if lastFlushHeight == tip.Height() {
			// Load validators from disk
			// Build out the nullifier map
			// We're good
			return nil
		} else if lastFlushHeight < tip.Height() {
			// Load the missing blocks from disk and
			// apply any changes to the accumulator.
			// Build the nullifier map.
			var (
				node = tip
				err  error
			)
			for {
				if node.height == lastFlushHeight+1 {
					break
				}
				node, err = tip.Parent()
				if err != nil {
					return err
				}

			}
			for {
				blk, err := node.Block()
				if err != nil {
					return err
				}
				for _, out := range blk.Outputs() {
					adb.acc.Insert(out.Commitment, false)
				}
				if node.height == tip.height {
					break
				}
				node, err = node.Child()
				if err != nil {
					return err
				}
			}
			if err := adb.Flush(flushRequired, tip.height); err != nil {
				return err
			}
		} else if lastFlushHeight > tip.Height() {
			// This really should never happen.
			// If we're here it's unlikely the tip node
			// has any attached children that we can use
			// to load the blocks and remove the changes
			// from the accumulator. Panic?
		}
	case scsFlushOngoing:
		// TODO: what should we do here? Let's hope this never
		// happens because at present we don't have any way to
		// rollback the accumulator. The only way to get the
		// accumulator to the current state is to recalculate
		// it from genesis.
	case scsEmpty:
		// New node. Grab genesis block and start applying
		// changes up to the tip.
		var (
			node = tip
			err  error
		)
		for {
			node, err = tip.Parent()
			if err != nil {
				return err
			}
			if node.height == 0 {
				break
			}
		}
		for {
			blk, err := node.Block()
			if err != nil {
				return err
			}
			for _, out := range blk.Outputs() {
				adb.acc.Insert(out.Commitment, false)
			}
			if node.height == tip.height {
				break
			}
			node, err = node.Child()
			if err != nil {
				return err
			}
		}
		if err := adb.Flush(flushRequired, tip.height); err != nil {
			return err
		}

	}
	return nil
}

func (adb *AccumulatorDB) Accumulator() *Accumulator {
	adb.mtx.RLock()
	defer adb.mtx.RUnlock()

	return adb.acc.Clone()
}

func (adb *AccumulatorDB) Commit(accumulator *Accumulator, chainHeight uint32, mode flushMode) error {
	adb.mtx.Lock()
	defer adb.mtx.Unlock()

	adb.acc = accumulator

	return adb.flush(mode, chainHeight)
}

func (adb *AccumulatorDB) Flush(mode flushMode, chainHeight uint32) error {
	adb.mtx.Lock()
	defer adb.mtx.Unlock()

	return adb.flush(mode, chainHeight)
}

func (adb *AccumulatorDB) flush(mode flushMode, chainHeight uint32) error {
	switch mode {
	case flushRequired:
		return adb.flushToDisk(chainHeight)
	case flushPeriodic:
		if adb.lastFlush.Add(maxTimeBetweenFlushes).Before(time.Now()) {
			return adb.flushToDisk(chainHeight)
		}
		return nil
	default:
		return fmt.Errorf("unsupported flushmode for the accumulator db")
	}
}

func (adb *AccumulatorDB) flushToDisk(chainHeight uint32) error {
	if err := dsPutAccumulatorConsistencyStatus(adb.ds, scsFlushOngoing); err != nil {
		return err
	}
	dbtx, err := adb.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())

	if err := dsPutAccumulator(dbtx, adb.acc); err != nil {
		return err
	}

	if err := dsPutAccumulatorLastFlushHeight(dbtx, chainHeight); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	if err := dsPutAccumulatorConsistencyStatus(adb.ds, scsConsistent); err != nil {
		return err
	}

	adb.lastFlush = time.Now()
	return nil
}
