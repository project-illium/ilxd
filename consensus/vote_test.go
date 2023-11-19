// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	mrand "math/rand"
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

	for i := 0; i < AvalancheFinalizationScore+10; i++ {
		assert.False(t, bc.RecordVote(blk1))
	}
	assert.True(t, bc.RecordVote(blk1))
	assert.True(t, bc.blockVotes[blk1].Status() == StatusFinalized)
	assert.True(t, bc.blockVotes[blk2].Status() == StatusNotPreferred)
}

func TestBlockChoiceWithConflicts(t *testing.T) {
	bc := NewBlockChoice(1)
	blka := randomBlockID()
	blkb := randomBlockID()
	blkc := randomBlockID()

	blka[0] = 0b00000000
	blkb[0] = 0b10000000
	blkc[0] = 0b11000000

	blks := []types.ID{blka, blkb, blkc}

	bc.AddNewBlock(blka, true)
	bc.AddNewBlock(blkb, true)
	bc.AddNewBlock(blkc, true)

	for i := 0; i < 10000; i++ {
		r := mrand.Intn(3)
		assert.False(t, bc.RecordVote(blks[r]))
	}

	assert.True(t, bc.bitRecord.activeBit > 0)
}
