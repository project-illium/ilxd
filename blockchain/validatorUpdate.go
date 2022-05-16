// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/types"
	"time"
)

type ValidatorEntry struct {
	addedNullifiers map[types.Nullifier]uint64
	spentNullifiers map[types.Nullifier]struct{}
	claimedCoinbase uint64
	epochBlocks     uint32
}

// ValidatorUpdate represents a set of changes to be applied
// to the ValidatorSet. The changes are merely staged here.
// This view needs to be committed to the ValidatorSet for the
// changes to apply.
type ValidatorUpdate struct {
	entries    map[peer.ID]*ValidatorEntry
	blockstamp time.Time
	blockHeight uint32
}

func (update *ValidatorUpdate) AddStake(validatorID peer.ID, amount uint64, nullifier types.Nullifier) {
	entry, ok := update.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[types.Nullifier]uint64),
			spentNullifiers: make(map[types.Nullifier]struct{}),
		}
	}
	entry.addedNullifiers[nullifier] = amount
}

func (update *ValidatorUpdate) SpendStake(validatorID peer.ID, nullifier types.Nullifier) {
	entry, ok := update.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[types.Nullifier]uint64),
			spentNullifiers: make(map[types.Nullifier]struct{}),
		}
	}
	entry.spentNullifiers[nullifier] = struct{}{}
}

func (update *ValidatorUpdate) RegisterCoinbase(validatorID peer.ID, amount uint64) {
	entry, ok := update.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[types.Nullifier]uint64),
			spentNullifiers: make(map[types.Nullifier]struct{}),
		}
	}
	entry.claimedCoinbase += amount
}

func (update *ValidatorUpdate) RegisterBlock(validatorID peer.ID) {
	entry, ok := update.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[types.Nullifier]uint64),
			spentNullifiers: make(map[types.Nullifier]struct{}),
		}
	}
	entry.epochBlocks++
}
