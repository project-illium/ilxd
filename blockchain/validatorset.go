// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mroth/weightedrand/v2"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"math/rand"
	"sync"
	"time"
)

const (
	// TODO: decide on a value for this
	maxTimeBetweenFlushes = time.Minute * 15
)

// setConsistencyStatus (SCS) codes are used to indicate the
// consistency status of the validator set state in the database.
type setConsistencyStatus uint8

const (
	// scsEmpty is used as a return value to indicate that no status was
	// stored.  The zero value should not be stored in the database.
	scsEmpty setConsistencyStatus = iota

	// scsConsistent indicates that the set is consistent with the last
	// flush hash stored in the database.
	scsConsistent

	// scsFlushOngoing indicates a flush is ongoing. If a node states with this
	// state it means it must have crashed in the middle of a flush.
	scsFlushOngoing

	// scsNbCodes is the number of valid consistency status codes.
	scsNbCodes
)

type Stake struct {
	Amount     uint64
	Blockstamp time.Time
}

type Validator struct {
	PeerID         peer.ID
	TotalStake     uint64
	Nullifiers     map[types.Nullifier]Stake
	unclaimedCoins uint64
	epochBlocks    uint32
	dirty          bool
}

type ValidatorSet struct {
	params       *params.NetworkParams
	ds           repo.Datastore
	validators   map[peer.ID]*Validator
	nullifierMap map[types.Nullifier]*Validator
	toDelete     map[peer.ID]struct{}
	chooser      *weightedrand.Chooser[peer.ID, uint64]
	dirty        bool
	lastFlush    time.Time
	mtx          sync.RWMutex
}

func NewValidatorSet(params *params.NetworkParams, ds repo.Datastore) *ValidatorSet {
	vs := &ValidatorSet{
		params:       params,
		ds:           ds,
		validators:   make(map[peer.ID]*Validator),
		nullifierMap: make(map[types.Nullifier]*Validator),
		toDelete:     make(map[peer.ID]struct{}),
		mtx:          sync.RWMutex{},
	}
	return vs
}

func (vs *ValidatorSet) Init(tip *blockNode) error {
	consistencyStatus, err := dsFetchValidatorSetConsistencyStatus(vs.ds)
	if err != nil {
		return err
	}
	lastFlushHeight, err := dsFetchValidatorLastFlushHeight(vs.ds)
	if err != nil {
		return err
	}

	switch consistencyStatus {
	case scsConsistent:
		validators, err := dsFetchValidators(vs.ds)
		if err != nil {
			return err
		}
		for _, val := range validators {
			vs.validators[val.PeerID] = val
			for nullifier := range val.Nullifiers {
				vs.nullifierMap[nullifier] = val
			}
		}
		if lastFlushHeight == tip.Height() {
			// Load validators from disk
			// Build out the nullifier map
			// We're good
			return nil
		} else if lastFlushHeight < tip.Height() {
			// Load validators from disk
			// Load the missing blocks from disk and
			// apply any changes to the validator set.
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

				validatorReward := uint64(0)

				if blk.Header.Height > 0 {
					parent, err := node.Parent()
					if err != nil {
						return err
					}
					prevHeader, err := parent.Header()
					if err != nil {
						return err
					}

					prevEpoch := (prevHeader.Timestamp - vs.params.GenesisBlock.Header.Timestamp) / vs.params.EpochLength
					blkEpoch := (blk.Header.Timestamp - vs.params.GenesisBlock.Header.Timestamp) / vs.params.EpochLength

					if blkEpoch > prevEpoch {
						validatorReward = calculateNextValidatorReward(vs.params, blkEpoch)
					}
				}

				if err := vs.CommitBlock(blk, validatorReward, flushPeriodic); err != nil {
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
			if err := vs.Flush(flushRequired, tip.height); err != nil {
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
		// Load the validators from disk
		// Iterate over all the blocks after lastFlushHeight
		// and remove any changes that may have been applied.
		// Traverse the blocks forward from lastFlushHeight to
		// the tip and apply the changes.
	case scsEmpty:
		// New node. Grab genesis block and start applying
		// changes to the validator set up to the tip.
	}

	b := make([]byte, 8)
	crand.Read(b)
	rand.Seed(int64(binary.BigEndian.Uint64(b)))

	choices := make([]weightedrand.Choice[peer.ID, uint64], 0, len(vs.validators))
	for peerID, validator := range vs.validators {
		choices = append(choices, weightedrand.NewChoice(peerID, validator.TotalStake))
	}
	// The chooser will panic if either:
	// - The weight is over the max (shouldn't happen as total illium coins fits in a uint64)
	// - The weight is zero (we will guard this in the method)
	vs.chooser, _ = weightedrand.NewChooser(choices...)

	return err
}

func (vs *ValidatorSet) GetValidator(id peer.ID) (*Validator, error) {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	val, ok := vs.validators[id]
	if !ok {
		return val, errors.New("not found")
	}
	cpy := &Validator{}
	copyValidator(cpy, val)
	return cpy, nil
}

func (vs *ValidatorSet) ValidatorExists(id peer.ID) bool {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	_, ok := vs.validators[id]
	return ok
}

func (vs *ValidatorSet) NullifierExists(nullifier types.Nullifier) bool {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	_, ok := vs.nullifierMap[nullifier]
	return ok
}

func (vs *ValidatorSet) TotalStaked() uint64 {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	return vs.totalStaked()
}

func (vs *ValidatorSet) totalStaked() uint64 {
	total := uint64(0)
	for _, val := range vs.validators {
		total += val.TotalStake
	}
	return total
}

// CommitBlock commits the changes to the validator set found in the block into the set.
// This function is fully atomic, if an error is returned, no changes are committed.
// It is expected that the block is fully validated before calling this method.
func (vs *ValidatorSet) CommitBlock(blk *blocks.Block, validatorReward uint64, flushMode flushMode) error {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	updates := make(map[peer.ID]*Validator)
	nullifiersToAdd := make(map[types.Nullifier]peer.ID)
	nullifiersToDelete := make(map[types.Nullifier]struct{})

	if blk.Header.Height > 0 {
		producerID, err := peer.IDFromBytes(blk.Header.Producer_ID)
		if err != nil {
			return err
		}

		blockProducer, ok := vs.validators[producerID]
		if !ok {
			return errors.New("block producer not found in validator set")
		}

		blockProducerNew := &Validator{}
		copyValidator(blockProducerNew, blockProducer)
		blockProducerNew.epochBlocks++

		updates[producerID] = blockProducerNew
	}

	for _, t := range blk.GetTransactions() {
		switch tx := t.GetTx().(type) {
		case *transactions.Transaction_CoinbaseTransaction:
			if blk.Header.Height > 0 {
				validatorID, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
				if err != nil {
					return err
				}

				valNew, ok := updates[validatorID]
				if !ok {
					valOld, ok := vs.validators[validatorID]
					if !ok {
						log.Warn("coinbase transaction for validator not in set")
					}
					valNew = &Validator{}
					copyValidator(valNew, valOld)
				}
				valNew.unclaimedCoins -= tx.CoinbaseTransaction.NewCoins
				updates[validatorID] = valNew
			}
		case *transactions.Transaction_StakeTransaction:
			validatorID, err := peer.IDFromBytes(tx.StakeTransaction.Validator_ID)
			if err != nil {
				return err
			}

			valNew, ok := updates[validatorID]
			if !ok {
				valOld, ok := vs.validators[validatorID]
				if !ok {
					valNew = &Validator{
						PeerID:         validatorID,
						TotalStake:     0,
						Nullifiers:     make(map[types.Nullifier]Stake),
						unclaimedCoins: 0,
						epochBlocks:    0,
						dirty:          true,
					}
				} else {
					valNew = &Validator{}
					copyValidator(valNew, valOld)
				}
			}
			valNew.TotalStake += tx.StakeTransaction.Amount
			valNew.Nullifiers[types.NewNullifier(tx.StakeTransaction.Nullifier)] = Stake{
				Amount:     tx.StakeTransaction.Amount,
				Blockstamp: time.Unix(blk.Header.Timestamp, 0),
			}
			updates[validatorID] = valNew
			nullifiersToAdd[types.NewNullifier(tx.StakeTransaction.Nullifier)] = validatorID
		case *transactions.Transaction_StandardTransaction:
			for _, nullifier := range tx.StandardTransaction.Nullifiers {
				valOld, ok := vs.nullifierMap[types.NewNullifier(nullifier)]
				if !ok {
					continue
				}
				v, ok := updates[valOld.PeerID]
				if ok {
					valOld = v
				}
				valNew := &Validator{}
				copyValidator(valNew, valOld)
				stake, ok := valNew.Nullifiers[types.NewNullifier(nullifier)]
				if !ok {
					return errors.New("nullifier not found with validator")
				}
				valNew.TotalStake -= stake.Amount
				delete(valNew.Nullifiers, types.NewNullifier(nullifier))
				nullifiersToDelete[types.NewNullifier(nullifier)] = struct{}{}
				updates[valNew.PeerID] = valNew
			}
		case *transactions.Transaction_MintTransaction:
			for _, nullifier := range tx.MintTransaction.Nullifiers {
				valOld, ok := vs.nullifierMap[types.NewNullifier(nullifier)]
				if !ok {
					continue
				}
				v, ok := updates[valOld.PeerID]
				if ok {
					valOld = v
				}
				valNew := &Validator{}
				copyValidator(valNew, valOld)
				stake, ok := valNew.Nullifiers[types.NewNullifier(nullifier)]
				if !ok {
					return errors.New("nullifier not found with validator")
				}
				valNew.TotalStake -= stake.Amount
				delete(valNew.Nullifiers, types.NewNullifier(nullifier))
				nullifiersToDelete[types.NewNullifier(nullifier)] = struct{}{}
				updates[valNew.PeerID] = valNew
			}
		}
	}

	for _, val := range updates {
		val.dirty = true
		vs.validators[val.PeerID] = val
	}

	for nullifier, peerID := range nullifiersToAdd {
		val := vs.validators[peerID]
		vs.nullifierMap[nullifier] = val
	}

	for nullifier := range nullifiersToDelete {
		delete(vs.nullifierMap, nullifier)
	}

	for _, val := range updates {
		if len(val.Nullifiers) == 0 {
			vs.toDelete[val.PeerID] = struct{}{}
			delete(vs.validators, val.PeerID)
		}
	}

	if validatorReward > 0 {
		totalStaked := vs.totalStaked()
		for _, val := range vs.validators {
			val.unclaimedCoins = validatorReward / (totalStaked / val.TotalStake)
		}
	}

	choices := make([]weightedrand.Choice[peer.ID, uint64], 0, len(vs.validators))
	for peerID, validator := range vs.validators {
		choices = append(choices, weightedrand.NewChoice(peerID, validator.TotalStake))
	}
	vs.chooser, _ = weightedrand.NewChooser(choices...)

	return vs.flush(flushMode, blk.Header.Height)
}

func (vs *ValidatorSet) WeightedRandomValidator() peer.ID {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	if vs.totalStaked() == 0 {
		return ""
	}

	return vs.chooser.Pick()
}

func (vs *ValidatorSet) Flush(mode flushMode, chainHeight uint32) error {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	return vs.flush(mode, chainHeight)
}

func (vs *ValidatorSet) flush(mode flushMode, chainHeight uint32) error {
	switch mode {
	case flushRequired:
		return vs.flushToDisk(chainHeight)
	case flushPeriodic:
		if vs.lastFlush.Add(maxTimeBetweenFlushes).Before(time.Now()) {
			return vs.flushToDisk(chainHeight)
		}
		return nil
	default:
		return fmt.Errorf("unsupported flushmode for the validator set")
	}
}

func (vs *ValidatorSet) flushToDisk(chainHeight uint32) error {
	if !vs.dirty {
		return nil
	}
	if err := dsPutValidatorSetConsistencyStatus(vs.ds, scsFlushOngoing); err != nil {
		return err
	}
	dbtx, err := vs.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())
	for _, val := range vs.validators {
		if val.dirty {
			if err := dsPutValidator(dbtx, val); err != nil {
				return err
			}
		}
	}
	for id := range vs.toDelete {
		if err := dsDeleteValidator(dbtx, id); err != nil {
			return err
		}
	}

	if err := dsPutValidatorLastFlushHeight(dbtx, chainHeight); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	if err := dsPutValidatorSetConsistencyStatus(vs.ds, scsConsistent); err != nil {
		return err
	}

	vs.toDelete = make(map[peer.ID]struct{})

	for _, val := range vs.validators {
		val.dirty = false
	}
	vs.dirty = false
	vs.lastFlush = time.Now()

	return nil
}

func copyValidator(dest *Validator, src *Validator) {
	dest.PeerID = src.PeerID
	dest.TotalStake = src.TotalStake
	dest.epochBlocks = src.epochBlocks
	dest.unclaimedCoins = src.unclaimedCoins
	dest.Nullifiers = make(map[types.Nullifier]Stake)
	for k, v := range src.Nullifiers {
		dest.Nullifiers[k] = v
	}
}
