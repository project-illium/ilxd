// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"math"
	"time"
)

// MainnetGenesisBlock is the genesis block for the mainnet.
//
// Technically this is not a valid block and would not pass the normal validation
// rules. The reason for this is because the genesis block needs to do two things
// ― create new coins and stake them. Without at least one validator created in the
// genesis block the chain cannot move forward. Normally however, a stake transaction
// cannot stake a coinbase created in the same block since the stake's zk-snark proof
// must make use of the block's txoRoot which isn't known until after the block is
// connected.
var MainnetGenesisBlock = blocks.Block{
	Header: &blocks.BlockHeader{
		Producer_ID: []byte{0x00}, //TODO
		Height:      0,
		Timestamp:   time.Unix(0, 0).Unix(), //TODO
		Parent:      make([]byte, 32),
		Version:     1,
		TxRoot:      []byte{0x00}, //TODO
		Signature:   []byte{0x00}, //TODO
	},
	Transactions: []*transactions.Transaction{
		{
			Tx: &transactions.Transaction_CoinbaseTransaction{
				CoinbaseTransaction: &transactions.CoinbaseTransaction{
					Validator_ID: []byte{0x00}, //TODO
					NewCoins:     math.MaxUint64 / 10,
					Outputs: []*transactions.Output{
						{
							Commitment:      []byte{0x00}, //TODO
							EphemeralPubkey: []byte{0x00}, //TODO
							Ciphertext:      []byte{0x00}, //TODO
						},
					},
					Signature: []byte{0x00}, //TODO
					Proof:     []byte{0x00}, //TODO
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: []byte{0x00}, //TODO
					Amount:       0,
					Nullifier:    []byte{0xff},
					Signature:    nil,
					Proof:        nil,
				},
			},
		},
	},
}
