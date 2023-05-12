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

// AccumulatorDB is responsible for persisting the accumulator on disk.
// It maintains a memory cache and periodically flushes to disk. This will
// speed up writes as we don't have to write to disk every block but
// rather only periodically.
type AccumulatorDB struct {
	acc       *Accumulator
	ds        repo.Datastore
	lastFlush time.Time
	mtx       sync.RWMutex
}

// NewAccumulatorDB returns a new AccumulatorDB
func NewAccumulatorDB(ds repo.Datastore) *AccumulatorDB {
	return &AccumulatorDB{
		acc: NewAccumulator(),
		ds:  ds,
		mtx: sync.RWMutex{},
	}
}

// Init will initialize the accumulator DB. It will load the accumulator
// from disk and if it is not currently at the tip of the chain it will
// roll the accumulator forward until it is up to the tip.
func (adb *AccumulatorDB) Init(tip *blockNode) error {
	consistencyStatus, err := dsFetchAccumulatorConsistencyStatus(adb.ds)
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
		adb.lastFlush = time.Now()
		adb.acc = acc
		if lastFlushHeight == tip.Height() {
			// We're good
			return nil
		} else if lastFlushHeight < tip.Height() {
			// Load the missing blocks from disk and
			// apply any changes to the accumulator.
			var (
				node = tip
				err  error
			)
			for {
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
				for _, out := range blk.Outputs() {
					// FIXME: the init function will ultimately need
					// to take in a list of commitments to protect.
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
			log.Fatal("AccumulatorDB last flush ahead of chain tip. Unable to repair. Use --reindexchainstate")
		}
	case scsFlushOngoing:
		// Let's hope this never happens because at present we
		// don't have any way to rollback the accumulator. The
		// only way to get the accumulator to the current state
		// is to recalculate it from genesis.
		log.Fatal("AccumulatorDB shut down mid flush. Unable to repair. Use --reindexchainstate")
	}
	return nil
}

// Accumulator returns a clone of the current accumulator.
func (adb *AccumulatorDB) Accumulator() *Accumulator {
	adb.mtx.RLock()
	defer adb.mtx.RUnlock()

	return adb.acc.Clone()
}

// Commit updates the accumulator in memory and flushes the change to disk using the flushMode.
// This commit is atomic. If there is an error flushing to the accumulator state in memory will
// not change.
func (adb *AccumulatorDB) Commit(accumulator *Accumulator, chainHeight uint32, mode flushMode) error {
	adb.mtx.Lock()
	defer adb.mtx.Unlock()

	if err := adb.flush(mode, accumulator, chainHeight); err != nil {
		return err
	}

	adb.acc = accumulator
	return nil
}

// Flush will trigger a manual flush of the accumulator DB to disk.
//
// This is safe for concurrent access
func (adb *AccumulatorDB) Flush(mode flushMode, chainHeight uint32) error {
	adb.mtx.Lock()
	defer adb.mtx.Unlock()

	return adb.flush(mode, adb.acc, chainHeight)
}

func (adb *AccumulatorDB) flush(mode flushMode, acc *Accumulator, chainHeight uint32) error {
	switch mode {
	case flushRequired:
		return adb.flushToDisk(acc, chainHeight)
	case flushPeriodic:
		if adb.lastFlush.Add(maxTimeBetweenFlushes).Before(time.Now()) {
			return adb.flushToDisk(acc, chainHeight)
		}
		return nil
	case flushNop:
		return nil
	default:
		return fmt.Errorf("unsupported flushmode for the accumulator db")
	}
}

func (adb *AccumulatorDB) flushToDisk(acc *Accumulator, chainHeight uint32) error {
	if err := dsPutAccumulatorConsistencyStatus(adb.ds, scsFlushOngoing); err != nil {
		return err
	}
	dbtx, err := adb.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())

	if err := dsPutAccumulator(dbtx, acc); err != nil {
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
