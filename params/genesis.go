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

var RegtestGenesisKey = []byte{
	0x08, 0x01, 0x12, 0x40, 0xbf, 0xd3, 0xc5, 0xae,
	0xbb, 0x0c, 0x88, 0xd3, 0x7a, 0x2f, 0xa7, 0xd7,
	0x09, 0x91, 0x85, 0xa9, 0x65, 0x5d, 0xcd, 0xdc,
	0x7a, 0xc2, 0x94, 0xef, 0x28, 0x15, 0xe9, 0xc1,
	0x41, 0xe9, 0x8a, 0x2e, 0x25, 0xfc, 0xc0, 0x58,
	0x01, 0x44, 0xb7, 0x49, 0xec, 0xfe, 0xf6, 0x90,
	0x91, 0xf9, 0x78, 0xe9, 0xa5, 0x91, 0xde, 0x9c,
	0xed, 0x30, 0xdc, 0x64, 0x89, 0x72, 0x2a, 0x3f,
	0x3d, 0xb3, 0xff, 0x16,
}

var RegtestViewKey = []byte{
	0x08, 0x04, 0x12, 0x40, 0xdc, 0xaa, 0xbf, 0x79,
	0x23, 0x43, 0x85, 0xa1, 0x3b, 0xda, 0x51, 0xa4,
	0xac, 0xca, 0x3b, 0x2a, 0xc6, 0x46, 0x7d, 0xed,
	0xc4, 0x4f, 0xff, 0x67, 0xc0, 0xef, 0x96, 0x20,
	0xb1, 0x2e, 0x3c, 0x69, 0xfd, 0x12, 0x64, 0xc9,
	0x2c, 0xc8, 0xd0, 0xb4, 0xfc, 0x8b, 0xdb, 0x81,
	0x1b, 0xbb, 0xc4, 0x87, 0x73, 0x49, 0x90, 0xc8,
	0x5c, 0xb4, 0x2e, 0x33, 0x3d, 0x6c, 0x86, 0xd9,
	0x5c, 0xe8, 0xb1, 0x6b,
}

var RegtestSpendKey = []byte{} // TODO

var RegtestGenesisBlock = &blocks.Block{
	Header: &blocks.BlockHeader{
		Version:   1,
		Height:    0,
		Parent:    hexToBytes("0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp: 0,
		TxRoot:    hexToBytes("e9a730339b6f38d83d7d53ef254f88ffd05fb531e56dc78f449bd4898d6269ca"),
	},
	Transactions: []*transactions.Transaction{
		{
			Tx: &transactions.Transaction_CoinbaseTransaction{
				CoinbaseTransaction: &transactions.CoinbaseTransaction{
					Validator_ID: hexToBytes("00240801122025fcc0580144b749ecfef69091f978e9a591de9ced30dc6489722a3f3db3ff16"),
					NewCoins:     230584300921369395,
					Outputs: []*transactions.Output{
						{
							Commitment: hexToBytes("924c38053f5f94baba523a83c10dcb94e75f642392e3094107c02d6d8c6c79ed"),
							Ciphertext: hexToBytes("83a4071ad9f1aa97189bc952d2e9a6a97f9c3077e94e7c4c9e612f865bb6e37ca963bb030509ca5d94212cea7231a433d044e2906dac5376f0ef3908904ed7bc77f54cefb93ac1293b3d074d5dd404dbf3e38288c4e6c82e9a07515ea180ed1481f4994bcde26cb1271042375299d08facaadd3e5ab567b064cb0e27e98b08641b9e30dec14cdf1284ebc483d1ec6b2fba2d001c7327f2b342c8dce3daa128c7080e27b407bc2ff5f7e6ae502c22729744080e2cf86a36bf84a87e56e77c88704c2f112184b80541e2caa309bb1d886ebc82b9adf641c5eb5a95d4f626002a6536df8c376bebae9d93093238135a2ba229ae295dd9e65ab462b7ef8b74e06de9f5975bf78304f809a133c4ece8e301b9fd5fa38318371a054c5e067c6a4c4d6a1e02b5f6e330ac4535596c099c50bd28"),
						},
					},
					Signature: hexToBytes("47d553bf0c2cd0b1cedcd1d83b3a007c1324d6184ea788bbe9afd532b020d8e06a9a6e391ae681a24e2da565877551f0278d1282c2cfb4ef8f955c77a21bde09"),
					Proof:     nil, // TODO
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("00240801122025fcc0580144b749ecfef69091f978e9a591de9ced30dc6489722a3f3db3ff16"),
					Amount:       230584300921369395,
					Nullifier:    hexToBytes("b178667bdbe651d9540bcc21e90a71360a474a5883b9eecc552004b75a681ec8"),
					TxoRoot:      hexToBytes("22a8ef3ef7e189bbc86b3e10569277db829442e5c8a30f407897842657c7b594"),
					Locktime:     0,
					Signature:    hexToBytes("3a78a3a0740fdc27f4462c27ecdebb4dee4debe7d56ce3bced914a4d85c5059f3f65d6ff23c9d752e6d3c438914ab48cb42232bab664c928a82ef60e22755401"),
					Proof:        nil, // TODO
				},
			},
		},
	},
}

func hexToBytes(s string) []byte {
	ret, _ := hex.DecodeString(s)
	return ret
}
