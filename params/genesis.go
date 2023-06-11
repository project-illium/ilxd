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
		TxRoot:      hexToBytes("90cab5a432f61ddd4eb709b2a2ef0bc92e370236a6936eafe5a56702310a6685"),
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
							Commitment: hexToBytes("029a245b7c6f614e1c7d84c06154a74f946ac8a3a2688a3c7ea537d833f12f94"),
							Ciphertext: hexToBytes("ad7a3d7f1d6020cc4d72fad09fe5f5840416c37693bf7779c36fc581e27eac67235a85454a098f6912876abe6fcaafb44ad6c6ef0f5d6873bda7fe4b49c5996dbe7cd1a87faad3ce7745a6fd95511c9dff38509b0f07f5da9b45f87f4d96bc2e1babf961ac07edc6b89c15137a8a99359a38c71441a9e831fb4762bff685860806618065ae09e1bcb6937486a4a7db3b0f4702b154736c1b58523560e18e1f0799189255b3593bdf5f435d0d969babdda82e073752bdce607aae34e26fb69debf825a8f85fb703c67f77d3c8182feadde8fdbc8611af40b001a6f424c65b7fee8ec47fefe0f675085eec6b99c2700881d881131ec15a24da938b1e3c0b2f5aaef3e7c59f8d538ce3894663a021756c924263894ecbfc630043929cabb5f6d44657874fd4714f8afc955f58a98c3c49c4"),
						},
					},
					Signature: hexToBytes("939e748b354ad3cab401b994f61b4b4dd0564879583302298a2f8553e232737df1c7a5ea5015dd049e81f01aa4d4ee51e1b3d9e2686d54442b717f45fd1ead0c"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       230584300921369395,
					Nullifier:    hexToBytes("931bf9e829315491463feed621c3b964663a12e05ac964c938b2e70bf75f14d0"),
					TxoRoot:      hexToBytes("aee276545429ddbb680c44289f240ab7a2ed784e5f7b3fcae03ab3ed880f655e"),
					Locktime:     0,
					Signature:    hexToBytes("53da3492794dd519eba67330489ba1e242a8c4639e34be2b1479c6ab9306c8916af05bca1a41ba1f9e34e57ecb03be306f0524f2d9337ee61647c775bbe3990a"),
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
