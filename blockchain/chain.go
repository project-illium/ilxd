// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"math"
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
	params        *params.NetworkParams
	ds            repo.Datastore
	index         *blockIndex
	accumulatorDB *AccumulatorDB
	validatorSet  *ValidatorSet
	nullifierSet  *NullifierSet
	txoRootSet    *TxoRootSet
	sigCache      *SigCache
	proofCache    *ProofCache

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

	prevHeader, err := b.index.Tip().Header()
	if err != nil {
		return err
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

	prevEpoch := (prevHeader.Timestamp - b.params.GenesisBlock.Header.Timestamp) / b.params.EpochLength
	blkEpoch := (blk.Header.Timestamp - b.params.GenesisBlock.Header.Timestamp) / b.params.EpochLength
	var validatorReward uint64
	if blkEpoch > prevEpoch {
		coinbase := calculateNextCoinbaseDistribution(b.params, blkEpoch)
		if err := dsIncrementCurrentSupply(dbtx, coinbase); err != nil {
			return err
		}

		treasuryCredit := coinbase / (100 / uint64(b.params.TreasuryPercentage))
		if err := dsCreditTreasury(dbtx, treasuryCredit); err != nil {
			return err
		}

		validatorReward = coinbase - treasuryCredit
	}
	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	b.index.ExtendIndex(blk.Header)

	// The following commits the changes to memory atomically so we don't need to worry about
	// rolling back the changes if the rest of this function errors. The only possible error is
	// an error flushing to disk, which we will just log. Any errors we should be able to repair
	// later.
	if err := b.validatorSet.CommitBlock(blk, validatorReward, flushPeriodic); err != nil {
		log.Errorf("Commit Block: Error flushing validator set: %s", err.Error())
	}

	if err := b.accumulatorDB.Commit(accumulator, blk.Header.Height, flushPeriodic); err != nil {
		log.Errorf("Commit Block: Error flushing accumulator: %s", err.Error())
	}

	// TODO: update indexers
	// TODO: notify subscribers of new block

	return nil
}

func calculateNextCoinbaseDistribution(params *params.NetworkParams, epoch int64) uint64 {
	if epoch > params.InitialDistributionPeriods {
		a := float64(params.TargetDistribution) * params.LongTermInflationRate
		return uint64(a * math.Pow(1.0+params.LongTermInflationRate, float64(epoch-params.InitialDistributionPeriods)))
	}

	return uint64((params.AValue * math.Pow(1.0-params.DecayFactor, float64(epoch))) + (float64(params.TargetDistribution) * (params.LongTermInflationRate * params.RValue)))
}

func calculateNextValidatorReward(params *params.NetworkParams, epoch int64) uint64 {
	coinbase := calculateNextCoinbaseDistribution(params, epoch)
	treasuryCredit := coinbase / (100 / uint64(params.TreasuryPercentage))
	return coinbase - treasuryCredit
}
