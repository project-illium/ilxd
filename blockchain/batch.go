// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"errors"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"sync"
)

// ErrMaxBatchSize is an error that means the batch exceeds the database limits
var ErrMaxBatchSize = errors.New("batch exceeds max batch size")

// Batch represents a batch of blocks that can be connected to the chain
// using only a single database transaction. This is faster than opening
// a new transaction for each individual block.
//
// This is used primarily during IBD.
type Batch struct {
	chain     *Blockchain
	blks      []*blocks.Block
	ops       int
	size      int
	committed bool
	blockWGs  map[types.ID]*sync.WaitGroup
}

// AddBlock adds a block to the batch. It will be added to the chain
// when commit is called.
//
// If the block would put the batch over the maximum ops or size limit
// for a database transaction an ErrMaxBatchSize is returned and the
// block is not added to the batch.
func (b *Batch) AddBlock(blk *blocks.Block) error {
	proofVal := NewProofValidator(b.chain.proofCache, b.chain.verifier)
	sigVal := NewSigValidator(b.chain.sigCache)

	wg := &sync.WaitGroup{}
	wg.Add(len(blk.Transactions) * 2)

	go func() {
		proofVal.Validate(blk.Transactions)
		wg.Done()
	}()
	go func() {
		sigVal.Validate(blk.Transactions)
		wg.Done()
	}()
	b.blockWGs[blk.ID()] = wg

	ops, size, err := datastoreTxnLimits(blk, 10)
	if err != nil {
		return err
	}
	if b.ops+ops > dsMaxBatchCount || b.size+size > dsMaxBatchSize {
		return ErrMaxBatchSize
	}
	b.ops += ops
	b.size += size
	b.blks = append(b.blks, blk)
	return nil
}

// Discard discards the current batch without committing anything
func (b *Batch) Discard() {
	if !b.committed {
		b.committed = true
		b.chain.stateLock.Unlock()
	}
}

// Commit will commit the batch to disk.
//
// Note this function is not fully atomic. To make it such we would need
// rollback ability on the validator set, nullifier set, accumulator db, txoroot db,
// and index (and possibly others).
//
// As it stands, we validate a block first. If it fails validation we commit all the prior
// blocks and return an error.
//
// If the database commit fails (which really should not happen), then the validator set
// and others could be left in a bad state.
func (b *Batch) Commit(flags BehaviorFlags) error {
	if b.committed {
		return errors.New("batch was discarded")
	}
	b.committed = true
	defer b.chain.stateLock.Unlock()

	dbtx, err := b.chain.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	flags = flags | BFBatchCommit | BFNoValidation

	defer dbtx.Discard(context.Background())
	for _, blk := range b.blks {
		b.blockWGs[blk.ID()].Wait()
		if _, err := b.chain.validateBlock(blk, flags); err != nil {
			if err := dbtx.Commit(context.Background()); err != nil {
				return err
			}
			return err
		}
		err = b.chain.connectBlock(dbtx, blk, flags)
		if err != nil {
			return err
		}
	}

	return dbtx.Commit(context.Background())
}
