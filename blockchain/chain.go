// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"errors"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain/indexers"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"math"
	"sync"
	"time"
)

const accumulatorCheckpointInterval = 100000

type flushMode uint8

const (
	// FlushRequired is used to signal that a validator set flush must take place.
	FlushRequired flushMode = iota

	// FlushPeriodic will flush if a certain time interval has passed since the last
	// flush.
	FlushPeriodic

	// FlushNop signals to skip the flush.
	FlushNop
)

// Blockchain is a class which handles all the functionality needed for maintaining
// the state of the chain. This includes validating blocks, connecting blocks to the
// chain and saving state to the database.
type Blockchain struct {
	params            *params.NetworkParams
	ds                repo.Datastore
	index             *blockIndex
	accumulatorDB     *AccumulatorDB
	validatorSet      *ValidatorSet
	nullifierSet      *NullifierSet
	txoRootSet        *TxoRootSet
	sigCache          *SigCache
	proofCache        *ProofCache
	indexManager      *indexers.IndexManager
	notifications     []NotificationCallback
	notificationsLock sync.RWMutex

	// stateLock protects concurrent access to the chain state
	stateLock sync.RWMutex
}

// NewBlockchain returns a fully initialized blockchain.
func NewBlockchain(opts ...Option) (*Blockchain, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	b := &Blockchain{
		params:            cfg.params,
		ds:                cfg.datastore,
		index:             NewBlockIndex(cfg.datastore),
		accumulatorDB:     NewAccumulatorDB(cfg.datastore),
		validatorSet:      NewValidatorSet(cfg.params, cfg.datastore),
		nullifierSet:      NewNullifierSet(cfg.datastore, cfg.maxNullifiers),
		txoRootSet:        NewTxoRootSet(cfg.datastore, cfg.maxTxoRoots),
		indexManager:      indexers.NewIndexManager(cfg.datastore, cfg.indexers),
		sigCache:          cfg.sigCache,
		proofCache:        cfg.proofCache,
		stateLock:         sync.RWMutex{},
		notificationsLock: sync.RWMutex{},
	}

	initialized, err := b.isInitialized()
	if err != nil {
		return nil, err
	}

	if !initialized {
		if err := dsInitTreasury(b.ds); err != nil {
			return nil, err
		}
		if err := dsInitCurrentSupply(b.ds); err != nil {
			return nil, err
		}
		if err := b.ConnectBlock(b.params.GenesisBlock, BFGenesisValidation); err != nil {
			return nil, err
		}
	} else {
		if err := b.index.Init(); err != nil {
			return nil, err
		}

		if err := b.accumulatorDB.Init(b.index.Tip()); err != nil {
			return nil, err
		}

		if err := b.indexManager.Init(b.index.Tip().Height(), b.GetBlockByHeight); err != nil {
			return nil, err
		}
	}
	if err := b.validatorSet.Init(b.index.Tip()); err != nil {
		return nil, err
	}
	return b, nil
}

// Close flushes all caches to disk and makes the node safe to shutdown.
func (b *Blockchain) Close() error {
	b.stateLock.Lock()
	defer b.stateLock.Unlock()

	tip := b.index.Tip()
	if err := b.validatorSet.Flush(FlushRequired, tip.height); err != nil {
		return err
	}
	return b.accumulatorDB.Flush(FlushRequired, tip.height)
}

// CheckConnectBlock checks that the block is valid for the current state of the blockchain
// and that it can be connected to the chain. This method does not change any blockchain
// state. It merely reads the current state to determine the block validity.
func (b *Blockchain) CheckConnectBlock(blk *blocks.Block) error {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	if err := b.checkBlockContext(blk.Header); err != nil {
		return err
	}

	return b.validateBlock(blk, BFNone)
}

// ConnectBlock attempts to connect the block to the chain. This method is atomic - if
// there is any error the state of the chain will be rolled back to the state prior to
// calling this method.
//
// The behavior flags can be used to control which aspects of the block are validated.
// Make sure the appropriate flags are set when calling this method as otherwise an
// invalid block could be connected.
func (b *Blockchain) ConnectBlock(blk *blocks.Block, flags BehaviorFlags) (err error) {
	b.stateLock.Lock()
	defer b.stateLock.Unlock()

	if !flags.HasFlag(BFGenesisValidation) {
		if err := b.checkBlockContext(blk.Header); err != nil {
			return err
		}
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
	if err := dsPutBlockIndexState(dbtx, &blockNode{blockID: blk.ID(), height: blk.Header.Height, timestamp: blk.Header.Timestamp}); err != nil {
		return err
	}

	if err := b.nullifierSet.AddNullifiers(dbtx, blk.Nullifiers()); err != nil {
		return err
	}

	accumulator := b.accumulatorDB.Accumulator()
	blockCointainsOutputs := false
	treasuryWidthdrawl := types.Amount(0)
	for _, tx := range blk.Transactions {
		for _, out := range tx.Outputs() {
			accumulator.Insert(out.Commitment, false)
			blockCointainsOutputs = true
		}
		if treasuryTx, ok := tx.Tx.(*transactions.Transaction_TreasuryTransaction); ok {
			treasuryWidthdrawl += types.Amount(treasuryTx.TreasuryTransaction.Amount)
		}
	}
	if treasuryWidthdrawl > 0 {
		if err := dsDebitTreasury(dbtx, treasuryWidthdrawl); err != nil {
			return err
		}
	}
	if blockCointainsOutputs {
		if err := b.txoRootSet.AddRoot(dbtx, accumulator.Root()); err != nil {
			return err
		}
	}

	var (
		validatorReward types.Amount
		newEpoch        bool
	)
	if flags.HasFlag(BFGenesisValidation) {
		if err := dsIncrementCurrentSupply(dbtx, types.Amount(blk.Transactions[0].GetCoinbaseTransaction().NewCoins)); err != nil {
			return err
		}
	} else {
		prevHeader, err := b.index.Tip().Header()
		if err != nil {
			return err
		}
		if b.params.Name == params.RegestParams.Name && prevHeader.Height == 0 {
			prevHeader.Timestamp = time.Now().Unix()
		}
		prevEpoch := (prevHeader.Timestamp - b.params.GenesisBlock.Header.Timestamp) / b.params.EpochLength
		blkEpoch := (blk.Header.Timestamp - b.params.GenesisBlock.Header.Timestamp) / b.params.EpochLength
		if blkEpoch > prevEpoch {
			coinbase := calculateNextCoinbaseDistribution(b.params, blkEpoch)
			if err := dsIncrementCurrentSupply(dbtx, coinbase); err != nil {
				return err
			}

			treasuryCredit := float64(coinbase) / (float64(100) / b.params.TreasuryPercentage)
			if err := dsCreditTreasury(dbtx, types.Amount(treasuryCredit)); err != nil {
				return err
			}

			validatorReward = coinbase - types.Amount(treasuryCredit)
			newEpoch = true
		}
	}

	if blk.Header.Height%accumulatorCheckpointInterval == 0 {
		if err := dsPutAccumulatorCheckpoint(dbtx, blk.Header.Height, accumulator); err != nil {
			return err
		}
	}

	if err := b.indexManager.ConnectBlock(dbtx, blk); err != nil {
		return err
	}
	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}
	// Now that we know the disk updated correctly we can update the cache. Ideally this would
	// be done in a commit hook, but that's a bigger change to the db interface.
	if blockCointainsOutputs {
		b.txoRootSet.UpdateCache(accumulator.Root())
	}

	b.index.ExtendIndex(blk.Header)

	// The following commits the changes to memory atomically so we don't need to worry about
	// rolling back the changes if the rest of this function errors. The only possible error is
	// an error flushing to disk, which we will just log. Any errors we should be able to repair
	// later.
	flushMode := FlushPeriodic
	if flags.HasFlag(BFNoFlush) {
		flushMode = FlushNop
	}
	if err := b.validatorSet.CommitBlock(blk, validatorReward, flushMode); err != nil {
		log.Errorf("Commit Block: Error flushing validator set: %s", err.Error())
	}

	if err := b.accumulatorDB.Commit(accumulator, blk.Header.Height, flushMode); err != nil {
		log.Errorf("Commit Block: Error flushing accumulator: %s", err.Error())
	}

	// Notify subscribers of new block.
	b.sendNotification(NTBlockConnected, blk)
	if newEpoch {
		b.sendNotification(NTNewEpoch, nil)
	}

	return nil
}

// ReindexChainState deletes all the state data from the database and rebuilds it
// by loading all blocks from genesis to the tip and re-processing them.
func (b *Blockchain) ReindexChainState() error {
	b.stateLock.Lock()
	defer b.stateLock.Unlock()

	dbtx, err := b.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())

	if err := dsDeleteAccumulator(dbtx); err != nil {
		return err
	}
	if err := dsDeleteAccumulatorCheckpoints(dbtx); err != nil {
		return err
	}
	if err := dsDeleteNullifierSet(dbtx); err != nil {
		return err
	}
	if err := dsDeleteTxoRootSet(dbtx); err != nil {
		return err
	}
	if err := dsDeleteBlockIndexState(dbtx); err != nil {
		return err
	}
	if err := dsDeleteValidatorSet(dbtx); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	if err := dsInitCurrentSupply(b.ds); err != nil {
		return err
	}
	if err := dsInitTreasury(b.ds); err != nil {
		return err
	}

	i := uint32(0)
	for {
		blockID, err := dsFetchBlockIDFromHeight(b.ds, i)
		if errors.Is(err, datastore.ErrNotFound) {
			break
		} else if err != nil {
			return err
		}

		blk, err := dsFetchBlock(b.ds, blockID)
		if err != nil {
			return err
		}

		flags := BFNoDupBlockCheck | BFFastAdd
		if i == 0 {
			flags = BFNoDupBlockCheck | BFFastAdd | BFGenesisValidation
		}

		if err := b.ConnectBlock(blk, flags); err != nil {
			return err
		}
		i++
	}

	return nil
}

// WeightedRandomValidator returns a validator weighted by their current stake.
func (b *Blockchain) WeightedRandomValidator() peer.ID {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return b.validatorSet.WeightedRandomValidator()
}

// BestBlock returns the ID, height, and timestamp of the block at the tip of the chain.
func (b *Blockchain) BestBlock() (types.ID, uint32, time.Time) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	tip := b.index.Tip()
	return tip.blockID, tip.height, time.Unix(tip.timestamp, 0)
}

// GetBlockByHeight returns the block at the given height. The block will be loaded from disk.
func (b *Blockchain) GetBlockByHeight(height uint32) (*blocks.Block, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	node, err := b.index.GetNodeByHeight(height)
	if err != nil {
		return nil, err
	}
	return node.Block()
}

// GetBlockByID returns the block with the given ID. The block will be loaded from disk.
func (b *Blockchain) GetBlockByID(blockID types.ID) (*blocks.Block, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	node, err := b.index.GetNodeByID(blockID)
	if err != nil {
		return nil, err
	}
	return node.Block()
}

// GetBlockIDByHeight returns the ID of the block at the given height.
func (b *Blockchain) GetBlockIDByHeight(height uint32) (types.ID, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	node, err := b.index.GetNodeByHeight(height)
	if err != nil {
		return types.ID{}, err
	}
	return node.ID(), nil
}

// GetHeaderByHeight returns the header at the given height. The header will be loaded from disk.
func (b *Blockchain) GetHeaderByHeight(height uint32) (*blocks.BlockHeader, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	node, err := b.index.GetNodeByHeight(height)
	if err != nil {
		return nil, err
	}
	return node.Header()
}

// HasBlock returns whether the block exists in the chain.
func (b *Blockchain) HasBlock(blockID types.ID) bool {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	_, err := b.index.GetNodeByID(blockID)
	return err == nil
}

// TreasuryBalance returns the current balance of the treasury.
func (b *Blockchain) TreasuryBalance() (types.Amount, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return dsFetchTreasuryBalance(b.ds)
}

// TxoRootExists returns whether the given root exists in the txo root set.
func (b *Blockchain) TxoRootExists(txoRoot types.ID) (bool, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return b.txoRootSet.RootExists(txoRoot)
}

// NullifierExists returns whether a nullifier exists in the nullifier set.
func (b *Blockchain) NullifierExists(n types.Nullifier) (bool, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return b.nullifierSet.NullifierExists(n)
}

// GetValidator returns the validator for the given ID
func (b *Blockchain) GetValidator(validatorID peer.ID) (*Validator, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	val, err := b.validatorSet.GetValidator(validatorID)
	if err != nil {
		return nil, err
	}
	ret := &Validator{}
	copyValidator(ret, val)
	return ret, nil
}

// GetAccumulatorCheckpointByTimestamp returns the accumulator checkpoint at or prior
// to the given timestamp.
// If there is no prior checkpoint and error will be returned.
func (b *Blockchain) GetAccumulatorCheckpointByTimestamp(timestamp time.Time) (*Accumulator, uint32, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	tip := b.index.Tip()

	priorCheckpoint := (tip.Height() / accumulatorCheckpointInterval) * accumulatorCheckpointInterval

	for {
		if priorCheckpoint <= 0 {
			return nil, 0, ErrNoCheckpoint
		}
		n, err := b.index.GetNodeByHeight(priorCheckpoint)
		if err != nil {
			return nil, 0, err
		}
		if timestamp.After(time.Unix(n.timestamp, 0)) {
			return b.getAccumulatorCheckpointByHeight(priorCheckpoint)
		}
		priorCheckpoint -= accumulatorCheckpointInterval
	}
}

// GetAccumulatorCheckpointByHeight returns the accumulator checkpoint at the given height.
// If there is no checkpoint at that height the prior checkpoint will be returned.
// If there is no prior checkpoint and error will be returned.
func (b *Blockchain) GetAccumulatorCheckpointByHeight(height uint32) (*Accumulator, uint32, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return b.getAccumulatorCheckpointByHeight(height)
}

func (b *Blockchain) getAccumulatorCheckpointByHeight(height uint32) (*Accumulator, uint32, error) {
	priorHeight := height / accumulatorCheckpointInterval
	if priorHeight == 0 {
		return nil, 0, ErrNoCheckpoint
	}
	acc, err := dsFetchAccumulatorCheckpoint(b.ds, priorHeight)
	if err != nil {
		return nil, 0, err
	}
	return acc, priorHeight, nil
}

// GetInclusionProof returns an inclusion proof for the input if the blockchain scanner
// had the encryption key *before* the commitment was processed in a block.
func (b *Blockchain) GetInclusionProof(commitment types.ID) (*InclusionProof, types.ID, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	proof, err := b.accumulatorDB.Accumulator().GetProof(commitment.Bytes())
	return proof, b.index.Tip().ID(), err
}

// Params returns the current chain parameters use by the blockchain.
func (b *Blockchain) Params() *params.NetworkParams {
	return b.params
}

// CurrentSupply returns the current circulating supply of coins.
func (b *Blockchain) CurrentSupply() (types.Amount, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	dbtx, err := b.ds.NewTransaction(context.Background(), true)
	if err != nil {
		return 0, err
	}
	defer dbtx.Discard(context.Background())

	supply, err := dsFetchCurrentSupply(dbtx)
	if err != nil {
		return 0, err
	}

	err = dbtx.Commit(context.Background())
	if err != nil {
		return 0, err
	}
	return supply, nil
}

// TotalStaked returns the total number of coins staked in the validator set.
func (b *Blockchain) TotalStaked() types.Amount {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return b.validatorSet.totalStaked()
}

// ValidatorSetSize returns the number of validators in the validator set.
func (b *Blockchain) ValidatorSetSize() int {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	return len(b.validatorSet.validators)
}

// Validators returns the full list of validators.
func (b *Blockchain) Validators() []*Validator {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	ret := make([]*Validator, 0, len(b.validatorSet.validators))
	for _, val := range b.validatorSet.validators {
		v := &Validator{}
		copyValidator(v, val)
		ret = append(ret, v)
	}
	return ret
}

// IsProducerUnderLimit returns whether the given validator is currently under the block production limit.
func (b *Blockchain) IsProducerUnderLimit(validatorID peer.ID) (bool, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	current, max, err := b.validatorSet.BlockProductionLimit(validatorID)
	if err != nil {
		return false, err
	}
	return current < max, nil
}

func (b *Blockchain) isInitialized() (bool, error) {
	_, err := dsFetchBlockIDFromHeight(b.ds, 0)
	if err == datastore.ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

/*func calculateNextCoinbaseDistributionOld(params *params.NetworkParams, epoch int64) types.Amount {
	if epoch > params.InitialDistributionPeriods {
		a := float64(params.TargetDistribution) * params.LongTermInflationRate
		return types.Amount(a * math.Pow(1.0+params.LongTermInflationRate, float64(epoch-params.InitialDistributionPeriods)))
	}

	return types.Amount((params.AValue * math.Pow(1.0-params.DecayFactor, float64(epoch))) + (float64(params.TargetDistribution) * (params.LongTermInflationRate * params.RValue)))
}*/

func calculateNextCoinbaseDistribution(params *params.NetworkParams, epoch int64) types.Amount {
	if epoch > params.InitialDistributionPeriods {
		a := float64(params.TargetDistribution) * params.LongTermInflationRate
		return types.Amount(a * math.Pow(1.0+params.LongTermInflationRate, float64(epoch-params.InitialDistributionPeriods)))
	}

	n := float64(params.TargetDistribution) - float64(params.GenesisBlock.Transactions[0].GetCoinbaseTransaction().NewCoins)
	periods := float64(params.InitialDistributionPeriods)
	w0 := (n / periods) * params.AValue
	wl := float64(params.TargetDistribution) * params.LongTermInflationRate
	r := math.Pow((wl / w0), (1 / periods))
	return types.Amount(w0 * math.Pow(r, float64(epoch)))
}

func calculateNextValidatorReward(params *params.NetworkParams, epoch int64) types.Amount {
	coinbase := calculateNextCoinbaseDistribution(params, epoch)
	treasuryCredit := float64(coinbase) * (params.TreasuryPercentage / 100)
	return coinbase - types.Amount(treasuryCredit)
}
