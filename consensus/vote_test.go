// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func randomBlockID() types.ID {
	b := make([]byte, 32)
	rand.Read(b)
	return types.NewID(b)
}

func TestBlockChoice(t *testing.T) {
	bc := NewBlockChoice(1)

	// blk1
	blk1 := randomBlockID()
	bc.AddNewBlock(blk1, true)
	pref := bc.GetPreference()

	assert.Equal(t, blk1, pref)
	assert.Len(t, bc.blockVotes, 1)
	assert.Equal(t, bc.bitRecord.isOnePreferred(), getBit(blk1, 0) == 1)

	// blk2
	blk2 := randomBlockID()
	bc.AddNewBlock(blk2, true)
	pref2 := bc.GetPreference()

	assert.NotEqual(t, blk2, pref2)
	assert.Len(t, bc.blockVotes, 2)
	assert.Equal(t, bc.bitRecord.isOnePreferred(), getBit(blk1, 0) == 1)

	for i := 0; i < FinalizationScore+11; i++ {
		_, ok := bc.RecordVote(blk1)
		assert.False(t, ok)
	}
	_, ok := bc.RecordVote(blk1)
	assert.True(t, ok)
	assert.True(t, bc.blockVotes[blk1].Status() == StatusFinalized)
	assert.True(t, bc.blockVotes[blk2].Status() == StatusNotPreferred)
}

func TestFlipping(t *testing.T) {
	bc := NewBlockChoice(1)

	blk1 := randomBlockID()
	blk2 := randomBlockID()
	blk1[0] = 0b00000000
	blk2[0] = 0b11111111
	bc.AddNewBlock(blk1, true)
	bc.AddNewBlock(blk2, true)
	bc.blockVotes[blk1].consider = 1
	bc.blockVotes[blk2].consider = 0

	// Preference is 0. Bit flips from 0 to 1.
	bc.bitRecord.confidence = 0b100111110
	bc.bitRecord.votes = math.MaxUint16
	bc.bitRecord.consider = math.MaxUint16

	bc.RecordVote(blk2)

	assert.Equal(t, blk2, bc.GetPreference())
	assert.Equal(t, uint8(0), bc.bitRecord.activeBit)

	// Preference is 0. Bit flips from 0 to 1.
	bc.bitRecord.confidence = 0b100111111
	bc.bitRecord.votes = math.MaxUint16
	bc.bitRecord.consider = math.MaxUint16

	bc.RecordVote(blk2)

	assert.Equal(t, blk2, bc.GetPreference())
	assert.Equal(t, uint8(1), bc.bitRecord.activeBit)
}
