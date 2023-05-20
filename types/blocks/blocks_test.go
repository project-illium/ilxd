// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blocks_test

import (
	"bytes"
	"encoding/json"
	"github.com/go-test/deep"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestJSONMarshalUnmarshal(t *testing.T) {
	b := proto.Clone(params.RegestParams.GenesisBlock)
	b.(*blocks.Block).Transactions = append(b.(*blocks.Block).Transactions, []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.StandardTransaction{
			Outputs: []*transactions.Output{
				{
					Commitment: bytes.Repeat([]byte{0xaa}, 10),
					Ciphertext: bytes.Repeat([]byte{0xbb}, 10),
				},
			},
			Nullifiers: [][]byte{bytes.Repeat([]byte{0x11}, 10)},
			TxoRoot:    bytes.Repeat([]byte{0x22}, 10),
			Locktime:   98,
			Fee:        5432,
			Proof:      bytes.Repeat([]byte{0x19}, 10),
		}),
		transactions.WrapTransaction(&transactions.MintTransaction{
			Type:         2,
			Asset_ID:     bytes.Repeat([]byte{0x12}, 10),
			DocumentHash: bytes.Repeat([]byte{0x34}, 10),
			NewTokens:    22,
			Outputs: []*transactions.Output{
				{
					Commitment: bytes.Repeat([]byte{0xcc}, 10),
					Ciphertext: bytes.Repeat([]byte{0xdd}, 10),
				},
			},
			Fee:        1234,
			Nullifiers: [][]byte{bytes.Repeat([]byte{0x33}, 10)},
			TxoRoot:    bytes.Repeat([]byte{0x44}, 10),
			MintKey:    bytes.Repeat([]byte{0x55}, 10),
			Locktime:   111,
			Signature:  bytes.Repeat([]byte{0x66}, 10),
			Proof:      bytes.Repeat([]byte{0x77}, 10),
		}),
		transactions.WrapTransaction(&transactions.TreasuryTransaction{
			Amount: 999,
			Outputs: []*transactions.Output{
				{
					Commitment: bytes.Repeat([]byte{0xee}, 10),
					Ciphertext: bytes.Repeat([]byte{0xff}, 10),
				},
			},
			ProposalHash: bytes.Repeat([]byte{0x88}, 10),
			Proof:        bytes.Repeat([]byte{0x99}, 10),
		}),
	}...)

	m, err := json.MarshalIndent(b, "", "    ")
	assert.NoError(t, err)

	var b2 blocks.Block
	err = json.Unmarshal(m, &b2)
	assert.NoError(t, err)

	assert.Empty(t, deep.Equal(b, proto.Clone(&b2)))
}
