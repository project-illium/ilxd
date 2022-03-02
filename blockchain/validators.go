// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"errors"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/repo"
	"sync"
)

type Validator struct {
	PeerID         peer.ID
	TotalStake     uint64
	Nullifiers     map[[32]byte]uint64
	unclaimedCoins uint64
	epochBlocks    uint32
	isModified     bool
}

type ValidatorSet struct {
	ds           repo.Datastore
	validators   map[peer.ID]*Validator
	nullifierMap map[[32]byte]*Validator
	toDelete     map[peer.ID]struct{}
	ms           *Multiset
	mtx          sync.RWMutex
}

func NewValidatorSet(ds repo.Datastore) (*ValidatorSet, error) {
	vs := &ValidatorSet{
		ds:           ds,
		validators:   make(map[peer.ID]*Validator),
		nullifierMap: make(map[[32]byte]*Validator),
		toDelete:     make(map[peer.ID]struct{}),
		mtx:          sync.RWMutex{},
	}
	return vs, nil
}

func (vs *ValidatorSet) Init() error {
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

func (vs *ValidatorSet) AddValidator(id peer.ID, stake uint64, nullifier [32]byte) error {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	_, exists := vs.validators[id]
	if exists {
		return errors.New("validator already exists in set")
	}

	val := &Validator{
		PeerID:     id,
		TotalStake: stake,
		Nullifiers: map[[32]byte]uint64{
			nullifier: stake,
		},
		isModified: true,
	}

	ser, err := serializeValidatorCommitment(val)
	if err != nil {
		return err
	}

	vs.validators[val.PeerID] = val
	for nullifier := range val.Nullifiers {
		vs.nullifierMap[nullifier] = val
	}

	vs.ms.Add(ser)

	if _, ok := vs.toDelete[id]; ok {
		delete(vs.toDelete, id)
	}
	return nil
}

func (vs *ValidatorSet) IncreaseStake(id peer.ID, amount uint64, nullifier [32]byte) error {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	val, ok := vs.validators[id]
	if !ok {
		return errors.New("validator not found")
	}

	_, ok = vs.nullifierMap[nullifier]
	if ok {
		return errors.New("nullifier already exists")
	}

	serializedCommitmentOld, err := serializeValidatorCommitment(val)
	if err != nil {
		return err
	}

	cpy := &Validator{}
	copyValidator(cpy, val)

	cpy.TotalStake += amount
	cpy.Nullifiers[nullifier] = amount
	cpy.isModified = true

	serializedCommitmentNew, err := serializeValidatorCommitment(cpy)
	if err != nil {
		return err
	}

	vs.nullifierMap[nullifier] = cpy
	vs.validators[cpy.PeerID] = cpy
	vs.ms.Remove(serializedCommitmentOld)
	vs.ms.Add(serializedCommitmentNew)

	return nil
}

func (vs *ValidatorSet) NullifierExists(nullifier [32]byte) bool {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	_, ok := vs.nullifierMap[nullifier]
	return ok
}

func (vs *ValidatorSet) SpendNullifier(nullifier [32]byte) error {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	validator, ok := vs.nullifierMap[nullifier]
	if !ok {
		return errors.New("nullifier not found in validator set")
	}

	serializedCommitmentOld, err := serializeValidatorCommitment(validator)
	if err != nil {
		return err
	}

	// We copy the validator first to keep things atomic
	// in case the serialization fails.
	cpy := &Validator{}
	copyValidator(cpy, validator)

	amount := cpy.Nullifiers[nullifier]
	cpy.TotalStake -= amount
	cpy.isModified = true
	delete(cpy.Nullifiers, nullifier)

	if len(cpy.Nullifiers) == 0 {
		delete(vs.validators, validator.PeerID)
		vs.toDelete[validator.PeerID] = struct{}{}
	} else {
		serializedCommitmentNew, err := serializeValidatorCommitment(cpy)
		if err != nil {
			return err
		}
		vs.ms.Add(serializedCommitmentNew)
	}
	vs.ms.Remove(serializedCommitmentOld)
	delete(vs.nullifierMap, nullifier)
	vs.validators[cpy.PeerID] = cpy
	return nil
}

func copyValidator(dest *Validator, src *Validator) {
	dest.PeerID = src.PeerID
	dest.TotalStake = src.TotalStake
	dest.epochBlocks = src.epochBlocks
	dest.unclaimedCoins = src.unclaimedCoins
	dest.Nullifiers = make(map[[32]byte]uint64)
	for k, v := range src.Nullifiers {
		dest.Nullifiers[k] = v
	}
}
