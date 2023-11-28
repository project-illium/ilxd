// Copyright (c) 2022 The illium developers
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
			expected:   "17fcbf74593838c62db86d114e1bab3bce2e09daab2eef410c0c6745b2eef696",
		},
		{
			expression: "1",
			expected:   "0a427bccbd5078eccd71ac1d575074dfd18a1402a698d1335d2af21767d0331c",
		},
		{
			expression: "555",
			expected:   "005ba8d537935281635f6eba89dda12ef51eb18d9e31a5d27add717b570d3bc6",
		},
	}

	for _, test := range tests {
		h, err := LurkCommit(test.expression)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, hex.EncodeToString(h))
	}
}
