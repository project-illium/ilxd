// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransactionScanner(t *testing.T) {
	priv, pub, err := crypto.GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	outputs := make([]*transactions.Output, 0, 10)
	for i := 0; i < 10; i++ {
		commitment := make([]byte, 32)
		rand.Read(commitment)
		cipherText := make([]byte, 1000)
		rand.Read(cipherText)

		if i < 5 {
			cipherText, err = pub.(*crypto.Curve25519PublicKey).Encrypt(cipherText)
			assert.NoError(t, err)
		}
		outputs = append(outputs, &transactions.Output{
			Commitment: commitment,
			Ciphertext: cipherText,
		})
	}

	scanner := NewTransactionScanner(priv.(*crypto.Curve25519PrivateKey))

	matches := scanner.ScanOutputs(&blocks.Block{
		Transactions: []*transactions.Transaction{
			{
				Tx: &transactions.Transaction_StandardTransaction{
					StandardTransaction: &transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							outputs[0],
							outputs[1],
						},
					},
				},
			},
			{
				Tx: &transactions.Transaction_StandardTransaction{
					StandardTransaction: &transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							outputs[2],
							outputs[3],
						},
					},
				},
			},
			{
				Tx: &transactions.Transaction_StandardTransaction{
					StandardTransaction: &transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							outputs[4],
							outputs[5],
						},
					},
				},
			},
			{
				Tx: &transactions.Transaction_StandardTransaction{
					StandardTransaction: &transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							outputs[6],
							outputs[7],
						},
					},
				},
			},
			{
				Tx: &transactions.Transaction_StandardTransaction{
					StandardTransaction: &transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							outputs[8],
							outputs[9],
						},
					},
				},
			},
		},
	})

	for i, out := range outputs {
		_, ok := matches[types.NewID(out.Commitment)]
		if i < 5 {
			assert.True(t, ok)
		} else {
			assert.False(t, ok)
		}
	}
}
