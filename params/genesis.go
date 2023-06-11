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
		TxRoot:      hexToBytes("543d9912019c74c0265659487afc3d999909b455af9c5eea1f43d05e448888ef"),
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
							Ciphertext: hexToBytes("fbf820b80761174146c58c3d26d30148af7ff75f387c8aca5a46190e29fe5c85d92a3b77321b5a386beae0ce958e2972785600d358d3e3700984be9496e88098cec56669583620e8afbd0d4ad210008cd3cf9b9793b2b4987665dcc856077f0c845ca7aa94e8236db5ec5d9e4272c43433b2f2a03af609e51c5a8ede18f044f88ad61da4bbc11d7230f1bed7178093ed865275078445a1cd4fdafb9d662c453b4efbc584c46851acf15ba58ccd0a8fee000e14270b327cb086526f2d96e2efad190ee188fa648351605c883785b8e3a59d1dc865b42eeade10e05ee15964f7dcee0aa0afd3b6c90b31aed73ba78f54c51059812d98cbe55a4c43be6bcda524798b0ebce073f1a7f7c575966eb0378fb554d93aec0c681c3fd292005864e8b0c250ac891421ab9ccb14ef167d69b9a36d\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
						{
							Commitment: hexToBytes("9cf23f7f270bfdbfe7913f1a14a84da61c961fa69b35da963602f0e38e027a17"),
							Ciphertext: hexToBytes("57e1672cc0c37b8c1f1d46c0051c04fb43f0fead44ff444f56603f596b35607d802c34cdc4f1bc1d9f05452d8045205f60f3937709f0f41102f2f47429967039b99e6c0842083b4c6253f50b9dc3c8173e51f7307ba0e45e98a0861cfcae2d9d167d6787e03302642d9c316e2bf516fafd3a733ae190ac132b6e5cf42f6917cf789cf9718b0617778a18fbc831e1da1f9a1d87339ca4dcbc75dce7a1b191e5bfc8087aebc05aa66010c674fbe4b8752a51201f2cc8d4e84cdb2773c313a6b3ed2be0607ff9f4d5372548bf8f3b783e63dda96fcebffbf5e5c78c4fb2c814915fbb353e59ca5856807b46244bf284bbd5c5d1ccd9ffe690ee2400e0a9e2bb0e1f7d012f9fe7c1c038e273628a51e7af85e3218a4ce9883fe78a16242ec65aa14d01c4d4cad90ea6a8ea3cc28144ce3d5b\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
					},
					Signature: hexToBytes("6316c248819f96fe66bcd927478d140eb3d687efc206b4d630357b4d89cd46f497f97f5dbee7d243ac3404f4ed68d7a9640f32444fb29292fd46ab47ffbeec0b"),
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
					TxoRoot:      hexToBytes("d9348297fbcc1a2aa1591701b6a749d253789b5ef97ef576b8b931a3093e07f9"),
					Locktime:     0,
					Signature:    hexToBytes("8707176c5eef8c6753d76459ebc3023413d2653693230a356f75cbd94c5dcbb7e7e87e450eaf3e4c01952b07f2c654a43d3e9b645d8511dc75c0f90d50572501"),
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
