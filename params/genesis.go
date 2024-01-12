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
		Timestamp:   1702087791,
		TxRoot:      hexToBytes("09ec970098ae317da212632582b5e56025725a0b95722910f9251b31c3d941b9"),
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
							Commitment: hexToBytes("2cd798c14226f7fa42cd97c73ff87d58bbb047cd229949347182c555855ff840"),
							Ciphertext: hexToBytes("afdd13937971c49d4c799bcae5718705b070577980fb89bc459f88460d704c6433435a909058cb95ef6be419d6daba3bc3d89b7224fdac4576a331ee2b6e9cde855ab6ecfd02de2e12a240f62657f7fea24407d5eb3bdd9ad1c543c366ae396eb65e0c8468c1a64d328ec90425418fd271a7672e588d83cd2a1cc5e1894d8ffad53772fdd4f6d9cde97b6eb50392f4658d7b303d12211519987f5555f6007e0dd3fa5935132b8e1eba4648959f2469a6e3e73320cca358190c26b17be0f45a0d7a40e49894cf22d8cca198b6f34070bc3fe6cc2710471d8ebe99125641f44545196f4fa5e445de74a165f52946d8b8ba14a41179b8c6a7acd385ad71c1186ae0abf0b7260a2aa29255af6e6c73e9010fd01ffc3282226e85"),
						},
						{
							Commitment: hexToBytes("1452a3ed9d0a0149590e1f8db5599899414c580fb0c9ebf3d501f78df525e0d4"),
							Ciphertext: hexToBytes("880de68a0e6b06f2729e40782984f890760ce1aa61d26065041ec6f3501df779b89dd48aa10d90967c4f8d2d6a0f14cb2f522f65bdfbead579254ad894f7e22a8d00e262f2d7065d2cdd3ed19eca649b0cbedf6d5b9828241d936fe9c2ecdabfa10030e921cc4d275fdc56aed3c8863e8429fd63e576ae5bd80df5b6fd97e16c3058c5c93b24abee38404be068e51676c899c024482ded25bc713ea41869fb24751542dfea556f5ff2aff915fd6a97189ed2cc526bb66501909f8612e47f6be95b6f5692e129e58926c01cc4b2809b145a5d4b0d406d6aeb48178879da0b00d57b7a42ba87cd56543df9ed7c87323b049e08e46cb4fc2794ba021a3695ae625403646833fb5fd449581c5f4332ea8a84defefef6473d393c"),
						},
					},
					Signature: hexToBytes("1cb809bb7a1b5556b4ab71e95758f2db3502595b55bff6dec9fa7412447fac77d4ffa1f4dd33c3043b77215e733e9866928edad7da8fd87d0ce1773d5098cd07"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("3e637dbcc806aacad18204c020aff8ba44e904a6208b8c3d5bdd115a896478f3"),
					TxoRoot:      hexToBytes("0a2547bd5cce134a02d2331246b3d7791451175d160ba7f59eaed8c5f29c3d9c"),
					LockedUntil:  0,
					Signature:    hexToBytes("bb2ff6d94ed10bdf4fe6f44c08abebe2a10bac6f2be343fc37e0f6506642e8a5a36976218f6bc8bf971c72923809240bc34c90a9fa7af8b9a79ec7c62dc2ee00"),
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
