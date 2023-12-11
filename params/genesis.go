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
		TxRoot:      hexToBytes("30fa83e7cb5354d5a75ebc84da78a7eda686668f76d5a7813d1d0afae7cabbc9"),
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
							Commitment: hexToBytes("2a2369a3c6a5add34f3f114747228cc8e091da46d266f3cbaab72d610287fc4a"),
							Ciphertext: hexToBytes("98d6940cc3956f5137b97035eafe369a74490cda8e8b3a812c7e1e1549509f35f08cc513c043df270645b4a273a2eec64056c40082751e1fa60cf7849c339259a0f86ab3956a378a53b11a96578b67837d717f33295366452f804229e3009b62e904d4e6ad1e744f1dae3361f26af1f12d715bdea3c178cc78190645dca9e902a9731c5285e0dccf1a1c1e8a41de71434a832e4b2499db10fe14fc958bc02da30158cb5d14500694c007fad2b35637bf5669b0bb6f74fe46cc04dc1c689269042a7b2045ba2dd60b3f3fe3f81fab39ed1758c4e63ffa718ddab75c7d84dcb00db2ceb0dbd28f76555b3f6941384af2237b86bb670fcff9f9aa0e35d7eef50b7dc337cde20fc5743d51c252b4ba7c12d3badecba762becab2"),
						},
						{
							Commitment: hexToBytes("2a2369a3c6a5add34f3f114747228cc8e091da46d266f3cbaab72d610287fc4a"),
							Ciphertext: hexToBytes("a131082f0f483a65732a850f0ab3fe773e2a5a87a6d63797914e019df1d65841e67843035cb4db44dc636b59540e195270b11bf88b655e5a41932141dea88b7cc33aaa6ffb958dcd6d45808e6e5f43634da9ea862e2165cdd0cefe36ec776a9dfdba1a79183af5c68d4d4338f09264b05a9d246174e72f789ce441222969213fe1212c23d11b846dc32543c77b20e40f6ee79d16af786924ce063d558dad2d35ae94ff2fc65dc8f48be5f0bd552f54f409509377652674bcdfd6fbea306ec3ba709eeeec4ae470a69cd163c31543a768e1996f7c6f18961bb8cda8f4e3e0e2f14b351e6ce3fe76690f86822f2fd22e33570edbb5afcfce3e35a255ed9d0c08a181c9fb6d0e6f8b92921e2044adb2d33411b1ee75af31bef3"),
						},
					},
					Signature: hexToBytes("edb7e2601bfe7ab001138358165fc94441b88629f9bfd9b5875f1759fad3e8fe3b14a53e7988f2bd68aef3e62834c2c70cfa51297360bfb447bcf24c20d1ba02"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("1585d657d64530ed83d9aff0a7311e3bb55baf3c13326f31742ded7861420845"),
					TxoRoot:      hexToBytes("38b150121ae0f03d305ace02b6b40363eeefa1605b0cb2657e7e1c52518a315e"),
					LockedUntil:  0,
					Signature:    hexToBytes("a249bd4133d2fbbe74888445dc1e988fd56ab8d150b22b962570b9dec835bb4234e6c03b9f2ac2e8b563c9ee102df343c3ad9a1edd34fea73a50ec43bec26307"),
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
