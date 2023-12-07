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
		TxRoot:      hexToBytes("7bfd0571b8cd10cdd271f5c481aa2cb65d8356955413b8bf010dd78ac59ef3c4"),
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
							Commitment: hexToBytes("08ca060d23d9e158958a3fb36c709b779a256e18c37c787a4625b59d1f3f1137"),
							Ciphertext: hexToBytes("0d4dd2ed11d5ea9020f8cd60907955e193eed9477de9ce7a8e20d7f2d1796242097004d71e15664eb362587a7165e073215ec05d76c8d97183ec202ad1433f8a438cb36366a7b49075066e235fc4d849e1e1f9b6dbae796030ba3f622cce196c6a57194e6a380f7988e14a36838ca03d5f8479d06bd8f287ea2e815282a792d8b53f349ff5bcb42197b8d976fa97f88f14dbbc79c0ec2fbe2e6fc0241fc459aa8977323b25e8603c4ce25535dce3032ef539e1220a2df7b5fa3c91d67bc61088b200f593d3c73c94ba6aee857c2c913b39ca5abcfda180285305cdf7aff1873b3a915492ed59233b43e157b081feea5834d9a97f39f9808dcc695f825d71015ff53f6768c5aad46894eb679e05d24611d79b5ed6e976f832"),
						},
						{
							Commitment: hexToBytes("5c2c85669ec171f213ed611f5679b49155f28f96fc6e2397ffa461dd9fe354c6"),
							Ciphertext: hexToBytes("8a4c970fbf53bbd7b0c1700ba8251f60e37ec913423817a2b90a0c79afd67a4f8da52f254ba78c1eff6b684ef602f13d71ed17a118aa8435d5f52c4a678d79d8e2653d084ac577247801bc786376ac067220ce839a0a29a5aafbcdddbddfd2202c5736b7e8e08037096b0f5f6455997fccdf43cd6341e3a3dfd5d73a4c9ad50e22769de01f5061476da461d575ab23d4616a7cf11d442e006fc2e70ad39f1cd74e9cc2f4ba02c7203941d6fd3a760592bf3dd82e20edd38e0b9aa05dbdcfc69f8b8cf9870d5ba687669cd8f187827bfa2b1e23f5796e42895dcada94eb646744e9ad1a9ce863f01a3ae6af7d2f8eeaabb89973f5b2da87fb2b3ad9033b5bf582e6bf64ee5fb1cdfdbfb7777a3fcbc2d38b216218ddfdd618"),
						},
					},
					Signature: hexToBytes("43b94ae2f963db02582464964ddfa66b9e1e56284caacfae47fb1d0d73cee713f5cac3bc4812f63b570aa66f62e85e7e9a911ce0fc1a20d1e6e5a5af0fb7190d"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("b8764530b4623284396426a0c29f27736b7cd0e4bcabcc843b10d06fb695e0e9"),
					TxoRoot:      hexToBytes("f8ccb983ae3ac8ddb21008148abe86ece89a5a41a2c8518f1bb20c17c6fd8e50"),
					LockedUntil:  0,
					Signature:    hexToBytes("96f034703fe6f5084d661f97f52d71b7721d37f6fdccc2c0c028b3d40d8011166b2f1c0e783eb030d8074c547776bf19b637dce1ad0f5c19e95dafa172ee9304"),
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
