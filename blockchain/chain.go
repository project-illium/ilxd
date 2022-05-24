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

	// flushIfNeeded will flush if the cache goes over the max size in memeory.
	flushIfNeeded
)

type Blockchain struct {
	ds                repo.Datastore
	index             *blockIndex
	accumulator       *Accumulator
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
			return AssertError(ErrDuplicateBlock)
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

	b.index.ExtendIndex(blk.Header)
	if err := b.index.Commit(dbtx); err != nil {
		return err
	}

	if err := b.nullifierSet.AddNullifiers(dbtx, blk.Nullifiers()); err != nil {
		return err
	}

	accumulatorCpy := *b.accumulator

	for _, t := range blk.Transactions {
		switch tx := t.GetTx().(type) {
		case *transactions.Transaction_StandardTransaction:
			for _, out := range tx.StandardTransaction.Outputs {
				accumulatorCpy.Insert(out.Commitment, false)
			}
		case *transactions.Transaction_CoinbaseTransaction:
			for _, out := range tx.CoinbaseTransaction.Outputs {
				accumulatorCpy.Insert(out.Commitment, false)
			}
		case *transactions.Transaction_MintTransaction:
			for _, out := range tx.MintTransaction.Outputs {
				accumulatorCpy.Insert(out.Commitment, false)
			}
		case *transactions.Transaction_TreasuryTransaction:
			for _, out := range tx.TreasuryTransaction.Outputs {
				accumulatorCpy.Insert(out.Commitment, false)
			}
		}
	}

	if err := b.txoRootSet.Add(dbtx, accumulatorCpy.Root()); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	if err := b.validatorSet.CommitBlock(blk, flushPeriodic); err != nil {
		return err
	}

	// TODO: commit accumulator to disk
	// TODO: update indexers
	// TODO: put the txoroot to the disk
	// TODO: commit changes to the treasury
	// TODO: commit balance changes if epoch
	// TODO: notify subscribers of new block

	return nil
}
