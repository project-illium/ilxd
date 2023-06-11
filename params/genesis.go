// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"encoding/hex"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
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
var MainnetGenesisBlock = &blocks.Block{
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
					NewCoins:     1 << 60 / 5,
					Outputs: []*transactions.Output{
						{
							Commitment: []byte{0x00}, //TODO
							Ciphertext: []byte{0x00}, //TODO
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

var RegtestMnemonicSeed = "machine owner oval voyage hero pride index rack doll planet route unaware survey canyon search million embrace power thumb goat design rich grab rhythm"

var RegtestGenesisKey = []byte{
	0x08, 0x01, 0x12, 0x40, 0xdc, 0xd8, 0xb1, 0x9d,
	0x2c, 0xc6, 0x6f, 0x0e, 0xc6, 0x13, 0xd4, 0xb0,
	0x8b, 0x7d, 0x73, 0x68, 0x2e, 0x2e, 0x11, 0x12,
	0x2c, 0x09, 0x95, 0x9a, 0x2c, 0xc0, 0x00, 0xb9,
	0x9a, 0x52, 0x5a, 0xcb, 0xb5, 0x62, 0xe4, 0x8c,
	0xa1, 0x18, 0xdb, 0x0f, 0x24, 0xa5, 0x3c, 0xfb,
	0xae, 0x9f, 0x6a, 0x3a, 0x67, 0xf8, 0x63, 0xe6,
	0x03, 0x15, 0x95, 0xd6, 0x43, 0xb7, 0xd8, 0x91,
	0x62, 0x1a, 0xc2, 0x80,
}

var RegtestSpendKey = []byte{} // TODO

var RegtestGenesisBlock = &blocks.Block{
	Header: &blocks.BlockHeader{
		Version:     1,
		Height:      0,
		Parent:      hexToBytes("0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:   0,
		TxRoot:      hexToBytes("04d7c694a5cd17834722a6e207ffaa8fe3e3c16aec6e0154b531890884c03957"),
		Producer_ID: nil,
		Signature:   nil,
	},
	Transactions: []*transactions.Transaction{
		{
			Tx: &transactions.Transaction_CoinbaseTransaction{
				CoinbaseTransaction: &transactions.CoinbaseTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					NewCoins:     230584300921369395,
					Outputs: []*transactions.Output{
						{
							Commitment: hexToBytes("afaeb38b759134f3c8d4c69ee81e5cc7874f9f969993f67874fd2714fd0020c5"),
							Ciphertext: hexToBytes("d7aa20b82fe77f8331da0afd9bccaf2b581b17a1ada0b3bf6f93dcc1d4472a9e279dc6ea01ab6eb1c4672d6a0ee4aa90f5d6d5e80175ce18998803fb948b92013eb2444c2722e551d4b189fd33b545050bb529d0d4335fd146f812f1583b876a6da65e99c84b300083df609699dd2f2d48c6fd727b0584e612eae769f098c3dafc0a795e66078bb74cfaee50030158a0b6e3c212ea44dedfbae3e3ed82f3cbb5eea7408234da015602cecfee48edc0b0c1f24f85cf1eb1d2541af0fc247d4a57431aabd1befb830cc9e96fdde9e963afffa5d691de3cbf904608ecdaae8bce474cc180b45b79c669b83af7d9e5dcc6b7f3e7497313594118f0ee79601fdab40b9343d7fc212bd161efbf82170e08b8be7541a7ba6fb5ce5636d8cace624bc62f8c3d0bd40deda92f5b76dae06a18baba"),
						},
					},
					Signature: hexToBytes("de9039bdecac1462ad45b15f9fc704afd7828815eb392b8e3a98161acdbca310fd23c4907277944a5c9dea2e0cb5034fedd090652e957a20e64f37f09a4ee902"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       230584300921369395,
					Nullifier:    hexToBytes("5affd8aa727c0147b9e2629ad2d5b6caac76af6129637a0e0b68485a3562574f"),
					TxoRoot:      hexToBytes("978a00564656c7c5a354c54825177d82711507a7c2cdce860a3bd3608f96e5b1"),
					Locktime:     0,
					Signature:    hexToBytes("dddeeadf486416c824e346ff73e28f200b795140bbaf6b92bd9c9573bc5868c7bb112e602fcf747fe87db7bce9916b2d6f63f1d6d4e465d245055444473b6106"),
					Proof:        []byte{},
				},
			},
		},
	},
}

func hexToBytes(s string) []byte {
	ret, _ := hex.DecodeString(s)
	return ret
}
