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

var RegtestGenesisBlock = &blocks.Block{
	Header: &blocks.BlockHeader{
		Version:     1,
		Height:      0,
		Parent:      hexToBytes("0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:   0,
		TxRoot:      hexToBytes("954aafa7bdc18ca3c1fb143023b56e1cadf51c79cce7bf26226b470f43230def"),
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
							Commitment: hexToBytes("8ad4cc17bcf1f0a3f3eda0754a92cb29e183b9cea74a9bb760e168ffab5d48a0"),
							Ciphertext: hexToBytes("1477a6e00cc34552d19b0c590da4f0869e0c856fc7dff5660eaa73d7cf69351c1ee4df6913129959fe3c41594eeb2ac8c76de368277f5070c00013b950781061846e475377c7d4ad98f20ca1a9bf5cfd18cf95f00badd3a882d0b21c743fa6e56425ceff907566108a84288a82ccb69b8e0f626505e2fe433812b62ae6fd0c3310e62605146aabeb9ef62d73c7db937a939a31f983e7d216fbd9a96cee1a897f924a6ff48377f6ac9ed96f58d36110b6a5bab4df593143c738a1dfc99361edbbe8c67c85c6bb69cfb3bb8a667dae20849571e334eeb0ce5015d53b398b66a76966f37211b5a149c4b748b7e366928fb87db392eaf6ce0a5dd0f20f08fb95f2bd4f73c27f09a1d17464aab389d2fd16e3fb14e4819b4dd225dd54d0686e66586b8ab9be42e2d91917d8da1f318b8d7534\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
						{
							Commitment: hexToBytes("484a548038929251daeda8be545ad912260fc1e6a9ec19d92d99368c48bd3dfd"),
							Ciphertext: hexToBytes("cb5c14651b303bcb934b5539de936b70bd772126f940b0aebe260c9bbe0f477c18f6c6e7aa11af679f90d455a1691c44fe8e5e39c320776fa091c602287210b1fc26031632920222be027858e194169c947898059cc248d35621af425f83f375e7e6e885612aaf9fd203a2d90bee7a55243bd452be56b288a48b8d36b1063fa6de2642e1e5163d65d32b57346213aea4276882182d68a4d14fd9d270dfa684e88372712310677751f2fb60d8a9d29fa2f3e9c7d578a86ed2793d20a8db3d8255cca192d4d97307638d45772a5e5cec277547786b5face8c75ebeae97e6e6ced7f316093c845068ccc8faa8165f02619e18ef0174c65f27ffebaf917968c3ba02e9eb2dc6c5acb0b01267cd424bd40be687b231b8ec6a0c33734e886021c15239256949a26b7c32b1714ded00cb72bbd4\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
					},
					Signature: hexToBytes("7b4c4d7c90ccc689e58b39293caba29b421aefbefe925765ec080e23fd40122cf2e8e06712055d5b2deeca1ddef7340942bd84337783e80f94a6a817528ab604"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("931bf9e829315491463feed621c3b964663a12e05ac964c938b2e70bf75f14d0"),
					TxoRoot:      hexToBytes("144512edae5997c684c1cb9a775ff64895974b66842f5470365727f548077d8b"),
					Locktime:     0,
					Signature:    hexToBytes("8342cd818c8651806b51b9283156af0cc8e05b9593684b0247612ba4fef47bc85d7f7381941c14ed849e038041626ff92f8b44656c03bf8621863ce01a3ff701"),
					Proof:        nil,
				},
			},
		},
	},
}

func hexToBytes(s string) []byte {
	ret, _ := hex.DecodeString(s)
	return ret
}
