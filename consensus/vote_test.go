// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVoteRecord(t *testing.T) {
	vr := NewBlockVoteRecord(types.ID{}, 0, true, true)

	for i := 0; i < 1000; i++ {
		assert.False(t, vr.regsiterVote(0x80))
		assert.Equal(t, StatusPreferred, vr.status())
	}

	for i := 0; i < AvalancheFinalizationScore+11; i++ {
		assert.False(t, vr.regsiterVote(1))
		assert.Equal(t, StatusPreferred, vr.status())
	}

	assert.True(t, vr.regsiterVote(1))
	assert.Equal(t, StatusFinalized, vr.status())

	vr.Reset(false)
	assert.Equal(t, StatusNotPreferred, vr.status())

	vr.Reject()
	assert.Equal(t, StatusRejected, vr.status())

	vr = NewBlockVoteRecord(types.ID{}, 0, true, false)
	for i := 0; i < 12; i++ {
		assert.False(t, vr.regsiterVote(1))
		assert.Equal(t, StatusNotPreferred, vr.status())
	}
	assert.True(t, vr.regsiterVote(1))

	vr = NewBlockVoteRecord(types.ID{}, 0, true, false)
	for i := 0; i < AvalancheFinalizationScore+100; i++ {
		assert.False(t, vr.regsiterVote(0))
		assert.Equal(t, StatusNotPreferred, vr.status())
	}
}
