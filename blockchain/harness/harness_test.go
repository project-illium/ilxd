// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTestHarness(t *testing.T) {
	h, err := NewTestHarness(DefaultOptions(), NTxsPerBlock(1))
	assert.NoError(t, err)

	err = h.GenerateBlocks(1000)
	assert.NoError(t, err)
}
