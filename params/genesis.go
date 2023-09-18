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
		Timestamp:   1695056535,
		TxRoot:      hexToBytes("92928a7201e0b517ce46e170b84197e59605addf51544cbf158d1283c00a13c3"),
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
							Commitment: hexToBytes("b7e6da4538fd77a02a6ebc3ed566e42267d39a1a65dd71c43962a00ff7ab4ebf"),
							Ciphertext: hexToBytes("fffd296af80f4ef0454eb7631c221447e3e1166f55fe3413bb0da38e1552c4649893d9a2125a4580c5e0feaae15315e80f62019cdb0f54c555c98bfa5600ff15cc7bbb99ee0cf55fd4797a259d78a1ecde4493664b0c351d10e4151c772bb5efe3d97605961c2c86535cff77801cdf6db79e663d50441b90d810196c0eac5622a7836e3025f3d8691e18267ea7daa6d0bf5b415ae6b4359969b9e726b4f1e8c528aaed8ed7a579949eb896a7b11c0c40c85cfcd6a22ebffaa2a543db807469ce181fdba8fc8a31e73d743f6e50651be4e36c4f8c425e17e4c02939951b75adc9384993143b835bf58650129740eb2385a7b80b8fcf554e967ac3d6eefcbd60e2c8f7a837cda202c292087d96ddb10c08e84f06cd8015c5d4\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
						{
							Commitment: hexToBytes("40af5b30bca59512b184a84472c8dc6c7948d28abc4cb7cfb97b9b829b24de0e"),
							Ciphertext: hexToBytes("c9636567b39563a9b1cda53b275e54e51de01d80cac663aa7c8bb2aeb793710dc95b1175c0bc75bbe211c55ea745049ca902c887a14b0c39283af1632967128c40fe99906cdc19bf7613bb20cc2c8f85229c7679c4725846709ce1f2384f6b47a3f65f2a5e35ed035fbb1dbc63a68bf42f460e396c260547842caa2da8b02c169290570d7dc2d7f0a745d7455956149c8c861f6d09f820ebf7b41855d96bbc91663e79bda1d34acaba79ef9607660db1c0def7c87f674c3bc6d753bdd2ffd67354215b50b320e54f6aeb35a639d69fd2e8b3835b021ac30d729638f63733ea191ae9bf5de4432fb57f93b092a005a5e0b38b366036d4b36c6e182e8ca6d01ddb5f9e8a3fac79cf804f4d80972931c2532a3f519944476f9a\n849e5f844b5e0ff9faae2fb9427d4a65dc6149f4a829b1d63e883156cde13a5499630218adbf280a2f1e355744d7af0b51e852087e6c6c373baa1a256853daaf6baacc3565cc175616339062826a0e03494ed94071ad5a9233b25d1a6ca61d48ab4d7f9d851a828f3e8"),
						},
					},
					Signature: hexToBytes("67be728cff30df54dc4e62675840a7462f29c5206fece906abe2c4e3108d84883a6b3ca6c73f66d573ed2dca6f80bc615d949e62162c1e7d1b0aae93f795c00d"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("0024080112208f743f9bd03eee12598ac062a71152033cb14252a4d31d87301afc09a590b825"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("c102fdffb718e5eb28b12ed5ecb14a492ed541bb1a101a0af0915191a4472f84"),
					TxoRoot:      hexToBytes("6431195243f28badfd69759f3c21162ae9bfef66b40defdb4d08b4e875258c78"),
					Locktime:     0,
					Signature:    hexToBytes("7bf460d8272c3c97abfca2d2ac8991b1bc996186b3aa3d8f142283f32506a603411a43aed241b809783d6d74afd99d20e3d35a5764215a083efbed3d87226508"),
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
		Timestamp:   1695056535,
		TxRoot:      hexToBytes("c7efd53ce446fb178b00158c16a3d99dc913d2e3d8063ef91025d65e7eb282bd"),
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
							Commitment: hexToBytes("73f751943d971fa316ce21ffb432ea6b1b88dc9c6b658f87ebeed648fab558c1"),
							Ciphertext: hexToBytes("d4d33bed8a94aa33974c5d99a6d9426eec610dc8ba5337d52e5c2e61a1c82774d0342890e264928b49909405cf6187e7f6142fa4e0fbeb6628eab7640ba5a2bb5840dbebadaa39234539e9b1f4804c3a66f09a16424a814463b58efa1bfa8eaf23f53b230eb8ee9c2afc67a5f6b02af5e1850101414fcbddab09d083c8de3eb4632a335063adb0aeda901a4a23a26d9173e88bec3636236103b114b02bd8bfe2e6c5ca49b10a51abbbbc34b9731af15c4e76126c9ba787f3820c33aac466bc9b2adb2852835e2b36cdcd414ff39f96c68e17d05f401ddc8f9f91150e2f6318ba19d97271790c5a0133e8a57f49c50dd63442f9d0bf488ec1682ea214db8cafd952b68bd90610de86482806c2090c70b090708999ee7c818f"),
						},
						{
							Commitment: hexToBytes("4d6143a6f4c054eda4c9159fe39928a0f12bdfc37b2d2b192528e247145372e4"),
							Ciphertext: hexToBytes("53942fd2961c3f9c8a02282e833020561a90a713fa6e51872c9ccc652a92d077413a5989f1c734077ba63ef64c1828e5be76050148d04ee9ce7de0c0a7135b5a22833790cf2c2aba873c3ea99e272b874395f55a66c6c3d37d42ed0d58205e4dfb6232058bd61c60e641e6d4725bf9bb67a86ef877994d2fe6b209e17e6da87bd8986d825df5551f1bf3951303b22ff90d0b857e644a6a2794c926a0dff47766687de8428298304dcf87017de648d769c0b0f7d2f0f705269ab6af577efb6fb9b4921dcb9e44f4057f6cce9a373e0b65bf042dbf9309eebbbc9ed0fb5ba9338dd89e83a81a327c0340a18d024a77307b9f957cfd705a7245cb590b1c2889041a4b326d7abb8ba9a91b9ac751df1856238eabf59c538978ad"),
						},
					},
					Signature: hexToBytes("abeecff94179aca3a90c7177bfbce79d2cffb67b4a15ad500ed410e64d897f302566c4a3769aa714993282efa8b68dabc55565335afdacbb4f144ac655a0b501"),
					Proof:     nil,
				},
			},
		},
		{
			Tx: &transactions.Transaction_StakeTransaction{
				StakeTransaction: &transactions.StakeTransaction{
					Validator_ID: hexToBytes("002408011220b562e48ca118db0f24a53cfbae9f6a3a67f863e6031595d643b7d891621ac280"),
					Amount:       115292150460684697,
					Nullifier:    hexToBytes("1d03d956bfafc66c2847637773464f0c0c64edc156daddc9dbba495b00c86199"),
					TxoRoot:      hexToBytes("51e914f765b37b7f26619ba8bfb7f8797898ca4b7296d617bcd1e760a0fa0a93"),
					Locktime:     0,
					Signature:    hexToBytes("ddadfe6e6976dad2622324930b62a5faab4eee3c0d4ef2405788fafb27680792ec53e67729bdf839bdec4c064582c7974827709f37c931eb814e4566947a6900"),
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
