// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStandardToExpr(t *testing.T) {
	n := make([]types.Nullifier, 0, 4)
	for i := 0; i < 4; i++ {
		r, err := types.RandomSalt()
		assert.NoError(t, err)
		n = append(n, r)
	}
	o := make([]PublicOutput, 0, 1)
	for i := 0; i < 1; i++ {
		r, err := types.RandomSalt()
		assert.NoError(t, err)
		out := PublicOutput{
			Commitment: types.NewID(r[:]),
			CipherText: make([]byte, 225),
		}
		rand.Read(out.CipherText)
		o = append(o, out)
	}
	s := StandardPublicParams{
		Nullifiers: n,
		Outputs:    o,
	}
	s.ToExpr()
}
