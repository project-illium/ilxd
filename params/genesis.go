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

var AlphanetGenesisBlock = &blocks.Block{
	Header: &blocks.BlockHeader{
		Version:     1,
		Height:      0,
		Parent:      hexToBytes("0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:   1698255320,
		TxRoot:      hexToBytes("dda191e07013fdaf63bde3bf9d3c36b4033d0a1b3f9fa10b5ed25840da98c7b2"),
		Producer_ID: nil,
		Signature:   nil,
	},
	Transactions: []*transactions.Transaction{
		{
			Tx: &transactions.Transaction_CoinbaseTransaction{
				CoinbaseTransaction: &transactions.CoinbaseTransaction{
					Validator_ID: hexToBytes("0024080112208f743f9bd03eee12598ac062a71152033cb14252a4d31d87301afc09a590b825"),
					NewCoins:     230584300921369395,
					Outputs: []*transactions.Output{
						{
							Commitment: hexToBytes("53dd3dee4a5f94338a9bfd836b4f8e9399269ecd3d2eafc6f423f3f41fb92b41"),
							Ciphertext: hexToBytes("42f3eb1a88c0bd608cd6d9e512216d6937a8d8dd39814b3de7ce6fb7a488536730bb5148b3258bc7598719b58405186ce81a6d1261380fae2412b67cfd56a6176a1aea10b71b250272080248aee82ed1d520f52c87b8233089cb78d04a0b25240e67f6a57ce08cf930c40d0353df63da0cee982baea8ad77f03ba097b8a4811eeb0edac8ddd0192f0036e4e3001df229559f3a6bcf2568489e280874dc6f2f9ce58201da0af57bdb9b2a3b1a2a69b4dbd04c1ec73c1242514802d9990c30a4ced69c46c27485d5e4fb4f61bfd3b40bf4071af421f4b3a9e3df5a2066ef1c677656be5c3dee32d9d7be861c3820549e66e5c124b405dcf80a3cfc359d6e7ff199a0aff768f60263b58556b772eec20c5486797ce8c3563d44\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
						{
							Commitment: hexToBytes("35b5cb47449714efe109600b8a219df48a2c7ecf3a75e9386b9020fd086cd632"),
							Ciphertext: hexToBytes("113717ff0868d5a45c46a5234337e85bf89022eeb937487464e43f8a6a5e4175375208e852a96c8f31bb3a7af41d72cb84ebd46383cd810337337addd01c471feb84fa8ea39d3201001c0a0eb70ba64ef174d0c12554104662d8c4d5d1b3d5f7c74d15364468c489c30670105e93958067aeabbab41999e3ce2d912fd0b428ec05ce4a4b8c2faf940c5560be5c761c163358cfd9acbf027913c0928d89f74f0a56dfbcf2f3c70de69d975f87a6417b9dbe467cc516a5e45a1f386fbf7b7d44d44a046240130288d8da32667e59c339ea531810875f29a49351326947fbaf04fdfc3a29743a4246f21e4d756dcb0dc3fa03afa74480b75ecfb0b8f4c5615a25591f2c07cb98e4cae4bda18f8be4b12bcaa738eeed14e6aea4\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
					},
					Signature: hexToBytes("9ff1a2f013bc89441feb5bc249a1a4bdcb643a9b485e702db2d4e5e599e3c77d6c93dfda25ebd8015185575ca765c5ce6a33ba04616145b31355601eafd20c03"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("0024080112208f743f9bd03eee12598ac062a71152033cb14252a4d31d87301afc09a590b825"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("e069e3ecde0c12582f76291675df9d58cace85966e278e502c884d710505b649"),
					TxoRoot:      hexToBytes("dc5f338c8e87938897bdcb2b5604c7360d08ce6c3bed599266cce36a141e8a85"),
					LockedUntil:  0,
					Signature:    hexToBytes("997315a3fb03d1dfaaeffc42766a6318563ad4908f6416189ebf441b279705262710ba15240fe152fff46647f32a700e888e755d7884ad43fc415ce596b1d304"),
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
		Timestamp:   1698255320,
		TxRoot:      hexToBytes("cad3be70918d25436316114fcea915383b44aa0583210d7942e9cc1267e128a5"),
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
							Commitment: hexToBytes("e1b34b86a90983e5ae56557b9257edbb2a7240fe7d7702cfa1db1b673de2fe48"),
							Ciphertext: hexToBytes("9e61b77f59e010207dee4889f7245fdc4a1c4951183a184d0af3acff58faa9154e7fb05435496435887b0ea49c35fbfd71e75d15caf252b9cfae7736592826ecfa742cb0f727ae27330af49f27de74d0bb6e7928c881de8071199d6a7db3adb962f1304b5d4b7be68af481e4fc1345471605e8d86f117c117f2294d272fca83b8ada3c23a627b0a07d4a3948e48634fdecb0a62516357e905cc5323650bf6dd0a6dcc4697ee303cd0bdc38156d8e75d896b530ac54f54b99734b2848538f7217ea480751e18e055599f2a17131c800cf0f4e462ea9356511e00169f87ebc5e319d22105e1853cc53c9578a9282c25861a1902242b14885304ad88657012a49c290ab25ca08575e75dc3b4a1c6785d15e722218d3d585aedb"),
						},
						{
							Commitment: hexToBytes("71d05c4eda5828403d70c606ec35d70ea0aead1aa27f41ac69e279f79f120475"),
							Ciphertext: hexToBytes("7069942b3a3dc89c07c559b6eda926f613c9fb1d84bac8376d770f887571e570f700b2951edfabaded64fdcb05b539ccbe0b45e1ad83f20b87bea60058f2903ed70b7f57019576361cdf4bf5fd8513bad457e866cd4f6af076cad6b8d70578ccc204ae38d1bc11f99a97d17e9330c38d67d39b3055444acd67bfdae6a2613ab97df8f7c7342655f56031de054b53dff3646ba38ce68e1b95e46865ee85bd81ba8d5064320a5aa05573ea413896e0bd45eed8959fbfaed43d40e15f155b6136bde5edecd40c9c116bef992f0b2299a85f05f123e59f917069a67f872e21a89b6afb1831e4a9d853a19479234bf33f48de7a1e7b94452c3f965132ffdaa76f005699c0167df8f6ae5f7f949ad0cd9ff877316cf750ba9083bf"),
						},
					},
					Signature: hexToBytes("03dd6039bed538a7b8d0c734479b861c81af88a49ea6a16ebb3d9ee134acd59771154dea2246565a9013b1d71db55d75a5c38cf0c3bc182acc5ce443737ac707"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("fb65e725f49bcc76c2ebbd74d128f24e9c2b69db4eb9eef356c5e26aea26cc79"),
					TxoRoot:      hexToBytes("2568b00c8a82bb0397549e31090e306977cf5f27bb2c0f9e993cfc09cbbc6943"),
					LockedUntil:  0,
					Signature:    hexToBytes("bd6ef129119afdfaffb793de2986e7c4a317f58247c0ab2140c6d780ac4702a4a3a324ec0e20adade6752d1eb3ae15ef64329e6bf18807fc0eb81c5f7436cd0d"),
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
