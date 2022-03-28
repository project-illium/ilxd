// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"errors"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/models"
	"github.com/project-illium/ilxd/repo"
	"sync"
	"time"
)

// Validator Set consistency status (VSCS) codes are used to indicate the
// consistency status of the validator set state in the database.
type vsConsistencyStatus uint8

const (
	// vscsEmpty is used as a return value to indicate that no status was
	// stored.  The zero value should not be stored in the database.
	vscsEmpty vsConsistencyStatus = 0

	// vscsConsistent indicates that the validator set is consistent with the
	// last flush hash stored in the database.
	vscsConsistent = iota

	// vscsFlushOngoing indicates a flush is ongoing. If a node states with this
	// state it means it must have crashed in the middle of a flush.
	vscsFlushOngoing

	// vscsNbCodes is the number of valid utxo consistency status codes.
	vscsNbCodes
)

type Stake struct {
	Amount     uint64
	Blockstamp time.Time
}

type Validator struct {
	PeerID         peer.ID
	TotalStake     uint64
	Nullifiers     map[models.Nullifier]Stake
	unclaimedCoins uint64
	epochBlocks    uint32
	dirty          bool
}

type ValidatorSet struct {
	ds           repo.Datastore
	validators   map[peer.ID]*Validator
	nullifierMap map[models.Nullifier]*Validator
	toDelete     map[peer.ID]struct{}
	mtx          sync.RWMutex
}

func NewValidatorSet(ds repo.Datastore) (*ValidatorSet, error) {
	vs := &ValidatorSet{
		ds:           ds,
		validators:   make(map[peer.ID]*Validator),
		nullifierMap: make(map[models.Nullifier]*Validator),
		toDelete:     make(map[peer.ID]struct{}),
		mtx:          sync.RWMutex{},
	}
	return vs, nil
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
	case vscsConsistent:
		if lastFlushHeight == tip.Height() {
			// Load validators from disk
			// Build out the nullifier map
			// We're good
		} else if lastFlushHeight < tip.Height() {
			// Load validators from disk
			// Load the missing blocks from disk and
			// apply any changes to the validator set.
			// Build the nullifier map.
		} else if lastFlushHeight > tip.Height() {
			// This really should never happen.
			// If we're here it's unlikely the tip node
			// has any attached children that we can use
			// to load the blocks and remove the validator
			// changes from the set. Panic?
		}
	case vscsFlushOngoing:
		// Load the validators from disk
		// Iterate over all the blocks after lastFlushHeight
		// and remove any changes that may have been applied.
		// Traverse the blocks forward from lastFlushHeight to
		// the tip and apply the changes.
	case vscsEmpty:
		// New node. Grab genesis block and start applying
		// changes to the validator set up to the tip.
	}
	return nil
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

func (vs *ValidatorSet) NullifierExists(nullifier models.Nullifier) bool {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	_, ok := vs.nullifierMap[nullifier]
	return ok
}

// Commit merges the view into the ValidatorSet.
func (vs *ValidatorSet) Commit(view *ValidatorViewpoint, flush bool) {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	modified := make(map[peer.ID]*Validator)
	addedNullifiers := make(map[models.Nullifier]*Validator)
	deletedNullifiers := make(map[models.Nullifier]struct{})
	for validatorID, entry := range view.entries {
		var newVal Validator
		validator, ok := vs.validators[validatorID]
		if !ok {
			newVal = Validator{
				PeerID:     validatorID,
				Nullifiers: make(map[models.Nullifier]Stake),
				dirty:      true,
			}
		} else {
			copyValidator(&newVal, validator)
		}
		for n, amount := range entry.addedNullifiers {
			newVal.Nullifiers[n] = Stake{
				Amount:     amount,
				Blockstamp: view.blockstamp,
			}
			newVal.TotalStake += amount
			addedNullifiers[n] = &newVal
		}
		for n := range entry.spentNullifiers {
			stake, ok := newVal.Nullifiers[n]
			if ok {
				newVal.TotalStake -= stake.Amount
				delete(newVal.Nullifiers, n)
				deletedNullifiers[n] = struct{}{}
			}
		}
		newVal.unclaimedCoins -= entry.claimedCoinbase
		newVal.epochBlocks += entry.epochBlocks
		newVal.dirty = true
		modified[validatorID] = &newVal
	}

	for id, valNew := range modified {
		if len(valNew.Nullifiers) == 0 {
			delete(vs.validators, id)
			vs.toDelete[id] = struct{}{}
		} else {
			vs.validators[id] = valNew
		}
	}

	for n, v := range addedNullifiers {
		vs.nullifierMap[n] = v
	}
	for n := range deletedNullifiers {
		delete(vs.nullifierMap, n)
	}
}

func (vs *ValidatorSet) flush() error {
	return nil
}

func copyValidator(dest *Validator, src *Validator) {
	dest.PeerID = src.PeerID
	dest.TotalStake = src.TotalStake
	dest.epochBlocks = src.epochBlocks
	dest.unclaimedCoins = src.unclaimedCoins
	dest.Nullifiers = make(map[models.Nullifier]Stake)
	for k, v := range src.Nullifiers {
		dest.Nullifiers[k] = v
	}
}
