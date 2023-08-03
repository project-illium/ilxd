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
		Timestamp:   1687122145,
		TxRoot:      hexToBytes("4ddee05d507f3a467cda0cd58438ec699855cf4331a17cbb691ba2742742dd8a"),
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
							Commitment: hexToBytes("a726eb3046c9bf71b5caad7ec607a3d480f8769c20a350fc9fdccc5990c6d6a4"),
							Ciphertext: hexToBytes("4d12d2df28cd05ab79555c2ec8c92121e507344cb088b50f09fcc9d4d129abfd51efcfe43d0ef860ac3c8ec7e0e2d9ac01a401cdd0ad5a7e07cd18d2358a020131b57d283fc0bf99ede24479f54dce266b1b241f6e6edf45514cbd2672739fdf0adac7b27df62af16c2544e563670a200cf38b4cb0dc0d7376648891935acdd40fded5ddb3d26bf28fd0705e70a10de995c19989f0b8dfde2c7124970508c9c93abd7e57d230d909f016ce09f950563153ffde7304f3feb31f1d74ccc88f38b3ad2e1b7b7bf4e7023ec00a7535e05fff8f0078e052ccc69bc7e2b1073eff5bcc262a2d6728e4e990b13ecb58e011f53154c832326369217858cec460aed0e56256a29bc6cedb962a4ee81e049315ff16203991dfa42854a59eb5f158302ac519cb0754e487e44e1fedf59ac6098e30ff\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
						{
							Commitment: hexToBytes("b45139694befbf32a292ff69dcc77750eb4307e38ce0d9843cac3b4eebf0ab3a"),
							Ciphertext: hexToBytes("c34309f0bc5f96b7a79f338b535e056cb5c2090d0fe7379a80232fdccd52802c822fe99d830ecdf584baaa3167a4646cb09c0fdfd758911d6c27cd9932f1f5661ad2c39db093e4fde76a9d57e51ba01933020689256247d807c6e933c03cc370aa7fad1579055003528d7e66f6913d1cbf63a5ae362f800578b31f3fee1685b5c4c2fcf2313aab715eaabfe63de7757c5ff7aff4ba5a8fb806d458bbaf1a888b9180e87834e2ee44f21ce2d0c7a2678eea66beeaf5aea6c57dea7ee627896eaa60d257b299cc1a5778037ebf69c7c79a440ce0aabbe0b308cea8ca847a7f3cc54ca321a79959346af88283fa1de9bc8975b45bb8b549533884dd20467a5b99667a94d051bdb07629b426759224d76a2e04a6e953835475021e022fd0e3982b5d6fc078db93553482e26379fd0bf4d74a\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
					},
					Signature: hexToBytes("ed4c4eec97e6ab445c5af269576ce5e63048797664a91b7eef944f7193a151fc2ea3b81188e3065f5468f34bccd158c0cbbf9e0b7501cf47084a30f25e443309"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("0024080112208f743f9bd03eee12598ac062a71152033cb14252a4d31d87301afc09a590b825"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("1d97d1d463483f23e40c5c0c59401502b0d4e8cf5e9502f31ced0c6b1211994b"),
					TxoRoot:      hexToBytes("47c000e919ce108860515ad155a7a2a3017578c67e9b2106e4b922eed34cbf13"),
					Locktime:     0,
					Signature:    hexToBytes("7e066f72273fe5116e4416e71beff23b7617b7d47c9c32e10b9b8f78f56820b2cd764c72381369e377757ed57dec30f14f80b48281f9f68caf0d22ff71acc002"),
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
		Timestamp:   1686874710,
		TxRoot:      hexToBytes("92d4855b2bf25530bcdf4898b8d64f4f5b2303351e5b48fdc1578cc18e0bb102"),
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
							Commitment: hexToBytes("affbfcda045fe785b71d1570196ee9a9a1a673199cf7f6bc2d0d6f6d7000c54b"),
							Ciphertext: hexToBytes("e56a9f590d2fab9ee66911b8dc088e679aabc202c02617aae35d08970ffa412d4dfb266004e161b869831a1a5c00be8914d73c562ba3812fb3d5e1775af729918d860db74cd9b5070946a971e93548bb5d8d42df160b6f7e531d175c9998b56da2ec61421cd506e2d8078b005869915cbcf009299d28075f89a12def87a35fc16b9bcfb88bee9c8dadd2304e4eb6a7b71e0542ff72d3f9cd08f36bcd8845a1a957941f21acc29a111525cd13e39f25c7434a71d821f78b204ea9649e108c6bf9e82abad730ef9d2fc9767e77de61e19391a1db11f9cb07d7afc2dda0270883dc8c0616d7fecbc4486c2b8416a5f6842ed81333948cfd91009ac689a9c02d4d8fed4775584ba14e11a62b827f46c9e42f4df759ed0f6c1061eeaf35e2a36ef91bbccb2d71da0205951455c469aafe1621\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
						{
							Commitment: hexToBytes("27838b298dea969c7975cb54a03e73a7a8c1ffe5b21f95d49df0c525e5961c0e"),
							Ciphertext: hexToBytes("612432dd6240c5d7ce0e36205205e8ba9c5e963d8f42401e22c7525f82fd834530a7d962d588a301f515f59521abe6faf652d7a47cd1c90bec6c4cbef5cc83f047a45c7ae189317900c25bce317ec91592e7cc64ef83b0b2193ea2e2a1d47df5694aaf7dface5bbf9c72cc1cb10195dc97bb42791dfd01cc76adca798fbec5040296714eebc26a994b7397d3f11974d947c5608fda195bc911f18e0ac2c02c6916c3f2c7814e36bb809b36498641a9f097fb759559b351251ad9fe0f498c7ab17e5855d211da92aeb414e0aaa8f61aafc5237bf13b48fa17eeaeaf5d3db70a6f8d056b9285d233c647e3276e090b2d18dffa912c4369d4a2fc0ec889074917387c94eba6d97f6939b10a215e680afe21e139499b8f9a0511f8f6c5d3afe14408650f1c7aa07adef46d35b18f7d9ce537\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
					},
					Signature: hexToBytes("91ea4c24122517a5eea0501fc404f24418ee2b843300d3d97088272d8931f24646741eb79ff075e8073bf6e8c5ea537ea20ae491488bced4c1f81e9cb5e97506"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("46432313f67a2c8888859ff50298f359ed9b3ce11854cadf6f4438138a72b0b1"),
					TxoRoot:      hexToBytes("63df89cd82d522372b0abc8990f673c2868486cdf6f674f54774a3866f66dcae"),
					Locktime:     0,
					Signature:    hexToBytes("094f78b6f02fb22cd5b4dda48b64a7aafdd88b39e0749ea0e7bac41fce883f831bc252f07c50a1725083831485c32254e32756994b3c10703f2c1ecaa0459506"),
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
