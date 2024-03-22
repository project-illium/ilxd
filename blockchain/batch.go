// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"errors"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/types/blocks"
	"sync"
)

// ErrMaxBatchSize is an error that means the batch exceeds the database limits
var ErrMaxBatchSize = errors.New("batch exceeds max batch size")

type batchWork struct {
	blk   *blocks.Block
	flags BehaviorFlags
	wg    *sync.WaitGroup
}

// Batch represents a batch of blocks that can be connected to the chain
// using only a single database transaction. This is faster than opening
// a new transaction for each individual block.
//
// This is used primarily during IBD.
type Batch struct {
	chain     *Blockchain
	ops       int
	size      int
	committed bool
	blockChan chan *batchWork
	wg        *sync.WaitGroup
	dbtx      datastore.Txn
	errResp   error
}

// AddBlock adds a block to the batch. It connects the block in a separate
// goroutine and returns immediately.
//
// If the block would put the batch over the maximum ops or size limit
// for a database transaction an ErrMaxBatchSize is returned and the
// block is not added to the batch.
func (b *Batch) AddBlock(blk *blocks.Block, flags BehaviorFlags) error {
	ops, size, err := datastoreTxnLimits(blk, 10)
	if err != nil {
		return err
	}
	if b.ops+ops > dsMaxBatchCount || b.size+size > dsMaxBatchSize {
		return ErrMaxBatchSize
	}

	proofVal := NewProofValidator(b.chain.proofCache, b.chain.verifier)
	sigVal := NewSigValidator(b.chain.sigCache)

	var wg *sync.WaitGroup
	if !(flags.HasFlag(BFFastAdd) || flags.HasFlag(BFNoValidation)) {
		wg = &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			proofVal.Validate(blk.Transactions)
			wg.Done()
		}()
		go func() {
			sigVal.Validate(blk.Transactions)
			wg.Done()
		}()
	}

	b.ops += ops
	b.size += size
	b.wg.Add(1)
	b.blockChan <- &batchWork{
		blk:   blk,
		flags: flags,
		wg:    wg,
	}
	return nil
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
func (b *Batch) Commit() error {
	if b.committed {
		return errors.New("batch was discarded")
	}
	b.wg.Wait()
	b.committed = true
	defer b.chain.stateLock.Unlock()
	close(b.blockChan)

	if err := b.dbtx.Commit(context.Background()); err != nil {
		return err
	}
	return b.errResp
}

func (b *Batch) handler() {
	for work := range b.blockChan {
		if b.committed {
			b.wg.Done()
			continue
		}
		if work.wg != nil {
			work.wg.Wait()
		}
		flags := work.flags | BFBatchCommit | BFNoValidation
		if _, err := b.chain.validateBlock(work.blk, flags); err != nil {
			b.errResp = fmt.Errorf("batch validate error at height: %d, err: %s", work.blk.Header.Height, err)
			b.wg.Done()
			go b.Commit()
			continue
		}
		err := b.chain.connectBlock(b.dbtx, work.blk, flags)
		if err != nil {
			b.errResp = fmt.Errorf("batch connect error at height: %d, err: %s", work.blk.Header.Height, err)
			b.wg.Done()
			go b.Commit()
			continue
		}
		b.wg.Done()
	}
}
