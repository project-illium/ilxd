// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLurkCommit(t *testing.T) {
	tests := []struct {
		expression string
		expected   string
	}{
		{
			expression: "(cons 1 (cons 2 nil))",
			expected:   "0c9e81a0ae6f34187171b5eed2e01438b5a5f7d16a25cbbe7edca785fea8be39",
		},
		{
			expression: "1",
			expected:   "07219e1a9376d53d13febc75310471e8cd865a1e17031f252aa09009a3d9387a",
		},
		{
			expression: "555",
			expected:   "289e3e85bd6fa8d8e136f645bf59c9c429063dd2452ce2dea7a9fbef043559ff",
		},
	}

	for _, test := range tests {
		h, err := LurkCommit(test.expression)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, hex.EncodeToString(h))
	}
}
