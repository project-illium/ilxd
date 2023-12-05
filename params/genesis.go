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
		Timestamp:   1701744150,
		TxRoot:      hexToBytes("a470a3c0e95660758f7f3ef1a34bf4c59bb468e10c52a8ebc1290cbbb3e40014"),
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
							Commitment: hexToBytes("e515e2a8bca20de7a75dfd3d879504bc4f9304a408ea7bb26271be2c107f73d8"),
							Ciphertext: hexToBytes("61d217be455ffd20ae908de9222656d8c94c7536c5ebf5dbb8d243aeddfdc07a3d2e2065ab71a6181899e8ba5c15af9e42c3c43e2b087e295d933c11a1ee2b68d90c23a1f02ab684e8e091995388bda611fb868268e21f5b19ad39307afdf58bf22827dc659009224adf2de0204a3695dfe1fd475b256963b7612ff5acafcca6da456104039e820f5bf308dc59de436f451948fa7a22cfe831165e3ad06fbb4268a860ff87f8ffa1420c9069e5c89d74c3d1e75901d2a04e0b76e04a8ef678ecac9527086c304bc52e69f2e3af7df7925e277e6981fda4332de2047ea3f309db8b51d03cdd0092437d78c45dfc869208882b372ea4a6ee9401043102a5dc3291956b7e262536a77bcb1549f82ca7854f858de3eda61f318c"),
						},
						{
							Commitment: hexToBytes("c7d0d493aee86501ed640839bef7f08510062e7effe70c91060156f641d0272e"),
							Ciphertext: hexToBytes("848f4ad0881910c447454953b4535373927ad5d20c2505ede1bd725b8c4d040959b7dc1ac310e74ccd61a4ddb72b0726de38d52e357e3489c438fe5c5f8d4bf0e496f0f26f453ee35d2eea0e0fb73d5ca6e0e94b483cf4c76fbb011aa5220b8d198e1ba01b02b0db0d8c734b8390f7b1a02a07de450462076758f50f57c4ebfb7ce3eedeec9fec68a314248af4c524c0c92a7cb68701a352c8e227ffe7fa0414314313b16c27f5a3e116d315c56b13a3a5016854e614adc18edaf4f26fde608778c6b75fa5809aea9100130f053c03eb9e50e671840b11fdcf9fee557e8b2fd8cfa44e6650a6a17fb4a02f5800b1b5cb02900b78b63c486d67ce6acadd6100dd564f92ab3dc5abc01ea16e2943cfaadfaf150b635a5ca00b"),
						},
					},
					Signature: hexToBytes("73a0746bae9c9425f244815bfe6995df3fbe898cbf55bb0d06f3d7510bff9e3e411e8e19cc22c1f100aec3033686ab099b32fdc8374ce227a53b246a60af7605"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("82664bfb796a5b3b4d8effe5b9b8a2664488a8b721dbe31678bcca46870b4753"),
					TxoRoot:      hexToBytes("59a3716efe80215b271fa9a9ff56d75953631773b384c264cd5532a9e3bb1edf"),
					LockedUntil:  0,
					Signature:    hexToBytes("1546ab9a49cbd32018646bc7d0e16ec3cd6e7e18eccfaa3e6e225ae26e81ca1bbc3f1ac72f3d8dfb7fa4352198d92b2207b17320ff902fe4b396bf6a9157f60f"),
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
