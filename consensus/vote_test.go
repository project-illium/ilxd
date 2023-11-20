// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
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

	for i := 0; i < AvalancheFinalizationScore+11; i++ {
		_, ok := bc.RecordVote(blk1)
		assert.False(t, ok)
	}
	_, ok := bc.RecordVote(blk1)
	assert.True(t, ok)
	assert.True(t, bc.blockVotes[blk1].Status() == StatusFinalized)
	assert.True(t, bc.blockVotes[blk2].Status() == StatusNotPreferred)
}
