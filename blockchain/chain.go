// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"sync"
)

type flushMode uint8

const (
	// flushRequired is used to signal that a validator set flush must take place.
	flushRequired flushMode = iota

	// flushPeriodic will flush if a certain time interval has passed since the last
	// flush.
	flushPeriodic
)

type Blockchain struct {
	ds                repo.Datastore
	index             *blockIndex
	accumulatorDB     *AccumulatorDB
	validatorSet      *ValidatorSet
	nullifierSet      *NullifierSet
	txoRootSet        *TxoRootSet
	sigCache          *SigCache
	proofCache        *ProofCache
	treasuryWhitelist map[types.ID]bool

	stateLock sync.RWMutex
}

func NewBlockchain(ds repo.Datastore) *Blockchain {
	return &Blockchain{}
}

func (b *Blockchain) CheckConnectBlock(blk *blocks.Block) error {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	if err := b.checkBlockContext(blk.Header); err != nil {
		return err
	}

	return b.validateBlock(blk, BFNone)
}

func (b *Blockchain) ConnectBlock(blk *blocks.Block, flags BehaviorFlags) (err error) {
	b.stateLock.Lock()
	defer b.stateLock.Unlock()

	if err := b.checkBlockContext(blk.Header); err != nil {
		return err
	}

	if !flags.HasFlag(BFNoDupBlockCheck) {
		exists, err := dsBlockExists(b.ds, blk.ID())
		if err != nil {
			return err
		}
		if exists {
			return ruleError(ErrDuplicateBlock, "duplicate block")
		}
	}

	if !flags.HasFlag(BFNoValidation) {
		if err := b.validateBlock(blk, flags); err != nil {
			return err
		}
	}

	dbtx, err := b.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())

	if err := dsPutBlock(dbtx, blk); err != nil {
		return err
	}
	if err := dsPutBlockIDFromHeight(dbtx, blk.ID(), blk.Header.Height); err != nil {
		return err
	}
	if err := dsPutBlockIndexState(dbtx, &blockNode{blockID: blk.ID(), height: blk.Header.Height}); err != nil {
		return err
	}

	if err := b.nullifierSet.AddNullifiers(dbtx, blk.Nullifiers()); err != nil {
		return err
	}

	accumulator := b.accumulatorDB.Accumulator()

	treasuryWidthdrawl := uint64(0)
	for _, t := range blk.Transactions {
		switch tx := t.GetTx().(type) {
		case *transactions.Transaction_StandardTransaction:
			for _, out := range tx.StandardTransaction.Outputs {
				accumulator.Insert(out.Commitment, false)
			}
		case *transactions.Transaction_CoinbaseTransaction:
			for _, out := range tx.CoinbaseTransaction.Outputs {
				accumulator.Insert(out.Commitment, false)
			}
		case *transactions.Transaction_MintTransaction:
			for _, out := range tx.MintTransaction.Outputs {
				accumulator.Insert(out.Commitment, false)
			}
		case *transactions.Transaction_TreasuryTransaction:
			treasuryWidthdrawl += tx.TreasuryTransaction.Amount
			for _, out := range tx.TreasuryTransaction.Outputs {
				accumulator.Insert(out.Commitment, false)
			}
		}
	}
	if treasuryWidthdrawl > 0 {
		if err := dsDebitTreasury(dbtx, treasuryWidthdrawl); err != nil {
			return err
		}
	}

	if err := b.txoRootSet.Add(dbtx, accumulator.Root()); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	b.index.ExtendIndex(blk.Header)

	// The following commits the changes to memory atomically so we don't need to worry about
	// rolling back the changes if the rest of this function errors. The only possible error is
	// an error flushing to disk, which we will just log. Any errors we should be able to repair
	// later.
	if err := b.validatorSet.CommitBlock(blk, flushPeriodic); err != nil {
		log.Errorf("Commit Block: Error flushing validator set: %s", err.Error())
	}

	if err := b.accumulatorDB.Commit(accumulator, blk.Header.Height, flushPeriodic); err != nil {
		log.Errorf("Commit Block: Error flushing accumulator: %s", err.Error())
	}

	// TODO: update indexers
	// TODO: commit unclaimed coins changes if epoch
	// TODO: notify subscribers of new block

	return nil
}
