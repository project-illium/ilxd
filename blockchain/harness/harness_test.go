// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTestHarness(t *testing.T) {
	_, err := NewTestHarness(DefaultOptions())
	assert.NoError(t, err)
}
