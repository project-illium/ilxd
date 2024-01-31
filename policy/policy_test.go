// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package policy

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFeePerKilobyte(t *testing.T) {
	tx := transactions.WrapTransaction(&transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: make([]byte, types.CommitmentLen),
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{make([]byte, 32)},
		TxoRoot:    make([]byte, 32),
		Fee:        20000,
		Proof:      make([]byte, 1000),
	})
	size, err := tx.SerializedSize()
	assert.NoError(t, err)
	kbs := float64(size) / 1000

	fpkb, ok, err := CalcFeePerKilobyte(tx)
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, types.Amount(float64(tx.GetStandardTransaction().Fee)/kbs), fpkb)
}
