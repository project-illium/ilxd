// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/models"
	"time"
)

type ValidatorEntry struct {
	addedNullifiers map[models.Nullifier]uint64
	spentNullifiers map[models.Nullifier]struct{}
	claimedCoinbase uint64
	epochBlocks     uint32
}

// ValidatorViewpoint represents a set of changes to be applied
// to the ValidatorSet. The changes are merely staged here.
// This view needs to be committed to the ValidatorSet for the
// changes to apply.
type ValidatorViewpoint struct {
	entries    map[peer.ID]*ValidatorEntry
	blockstamp time.Time
}

func (view *ValidatorViewpoint) AddStake(validatorID peer.ID, amount uint64, nullifier models.Nullifier) {
	entry, ok := view.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[models.Nullifier]uint64),
			spentNullifiers: make(map[models.Nullifier]struct{}),
		}
	}
	entry.addedNullifiers[nullifier] = amount
}

func (view *ValidatorViewpoint) SpendStake(validatorID peer.ID, nullifier models.Nullifier) {
	entry, ok := view.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[models.Nullifier]uint64),
			spentNullifiers: make(map[models.Nullifier]struct{}),
		}
	}
	entry.spentNullifiers[nullifier] = struct{}{}
}

func (view *ValidatorViewpoint) RegisterCoinbase(validatorID peer.ID, amount uint64) {
	entry, ok := view.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[models.Nullifier]uint64),
			spentNullifiers: make(map[models.Nullifier]struct{}),
		}
	}
	entry.claimedCoinbase += amount
}

func (view *ValidatorViewpoint) RegisterBlock(validatorID peer.ID) {
	entry, ok := view.entries[validatorID]
	if !ok {
		entry = &ValidatorEntry{
			addedNullifiers: make(map[models.Nullifier]uint64),
			spentNullifiers: make(map[models.Nullifier]struct{}),
		}
	}
	entry.epochBlocks++
}
