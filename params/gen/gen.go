// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	params2 "github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/walletlib"
	"log"
	"strings"
)

type GenerationParams struct {
	Mnemonic     string `long:"mnemonicseed" description:"The mnemonic seed to use when creating the params"`
	InitialCoins uint64 `long:"initialcoins" description:"The number of coins created by the genesis block"`
	Timestamp    int64  `long:"timestamp" description:"The genesis block timestamp"`
	NetParams    string `long:"params" description:"The network params to use: [mainnet, testnet1, regtest, alphanet]"`
}

// --timestamp=1698255320
// --initialcoins=230584300921369395

func main() {
	var params GenerationParams
	parser := flags.NewNamedParser("genesis generator", flags.Default)
	parser.AddGroup("Generation Options", "Options for generating a new genesis block", &params)
	if _, err := parser.Parse(); err != nil {
		log.Fatal(err)
	}

	var netParams *params2.NetworkParams
	switch strings.ToLower(params.NetParams) {
	case "mainnet":
		netParams = &params2.MainnetParams
	case "testnet1":
		netParams = &params2.Testnet1Params
	case "regtest":
		netParams = &params2.RegestParams
	case "alphanet":
		netParams = &params2.AlphanetParams
	default:
		log.Fatal("unknown net params")
	}

	kc, err := walletlib.NewKeychain(mock.NewMapDatastore(), netParams, params.Mnemonic)
	if err != nil {
		log.Fatal(err)
	}

	validatorKey, err := kc.NetworkKey()
	if err != nil {
		log.Fatal(err)
	}

	validatorID, err := peer.IDFromPrivateKey(validatorKey)
	if err != nil {
		log.Fatal(err)
	}
	validatorIDBytes, err := validatorID.Marshal()
	if err != nil {
		log.Fatal(err)
	}

	keys, err := kc.PrivateKeys()
	if err != nil {
		log.Fatal(err)
	}

	var (
		spendKey     crypto.PrivKey
		viewKey      crypto.PrivKey
		stakeAmt     = params.InitialCoins / 2
		secondOutAmt = params.InitialCoins - stakeAmt
	)

	for k := range keys {
		spendKey = k.SpendKey()
		viewKey = k.ViewKey()
		break
	}
	x, y := spendKey.GetPublic().(*icrypto.NovaPublicKey).ToXY()

	basicTransferCommitment, err := zk.LurkCommit(zk.BasicTransferScript())
	if err != nil {
		log.Fatal(err)
	}

	unlockingScript := types.UnlockingScript{
		ScriptCommitment: basicTransferCommitment,
		ScriptParams:     [][]byte{x, y},
	}
	scriptHash, err := unlockingScript.Hash()
	if err != nil {
		log.Fatal(err)
	}

	note0 := types.SpendNote{
		ScriptHash: scriptHash.Bytes(),
		Amount:     types.Amount(stakeAmt),
		AssetID:    types.IlliumCoinID,
	}
	salt, err := types.RandomSalt()
	if err != nil {
		log.Fatal(err)
	}
	note0.Salt = salt

	note1 := types.SpendNote{
		ScriptHash: scriptHash.Bytes(),
		Amount:     types.Amount(secondOutAmt),
		AssetID:    types.IlliumCoinID,
	}
	salt1, err := types.RandomSalt()
	if err != nil {
		log.Fatal(err)
	}
	note1.Salt = salt1

	ser, err := note0.Serialize()
	if err != nil {
		log.Fatal(err)
	}
	ciphertext0, err := viewKey.GetPublic().(*icrypto.Curve25519PublicKey).Encrypt(ser)
	if err != nil {
		log.Fatal(err)
	}

	ser, err = note1.Serialize()
	if err != nil {
		log.Fatal(err)
	}
	ciphertext1, err := viewKey.GetPublic().(*icrypto.Curve25519PublicKey).Encrypt(ser)
	if err != nil {
		log.Fatal(err)
	}

	nullifier, err := types.CalculateNullifier(0, note0.Salt, unlockingScript.ScriptCommitment, unlockingScript.ScriptParams...)
	if err != nil {
		log.Fatal(err)
	}

	commitment0, err := note0.Commitment()
	if err != nil {
		log.Fatal(err)
	}
	commitment1, err := note1.Commitment()
	if err != nil {
		log.Fatal(err)
	}

	blk := &blocks.Block{
		Header: &blocks.BlockHeader{
			Version:     1,
			Height:      0,
			Parent:      make([]byte, 32),
			Timestamp:   params.Timestamp,
			TxRoot:      nil,
			Producer_ID: nil,
			Signature:   nil,
		},
		Transactions: []*transactions.Transaction{
			transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: validatorIDBytes,
				NewCoins:     params.InitialCoins,
				Outputs: []*transactions.Output{
					{
						Commitment: commitment0.Bytes(),
						Ciphertext: ciphertext0,
					},
					{
						Commitment: commitment1.Bytes(),
						Ciphertext: ciphertext1,
					},
				},
				Signature: nil,
				Proof:     nil, // TODO
			}),
			transactions.WrapTransaction(&transactions.StakeTransaction{
				Validator_ID: validatorIDBytes,
				Amount:       stakeAmt,
				Nullifier:    nullifier.Bytes(),
				TxoRoot:      nil,
				Signature:    nil,
				Proof:        nil, // TODO
			}),
		},
	}

	sigHash, err := blk.Transactions[0].GetCoinbaseTransaction().SigHash()
	if err != nil {
		log.Fatal(err)
	}
	blk.Transactions[0].GetCoinbaseTransaction().Signature, err = validatorKey.Sign(sigHash)
	if err != nil {
		log.Fatal(err)
	}

	acc := blockchain.NewAccumulator()
	acc.Insert(blk.Transactions[0].GetCoinbaseTransaction().Outputs[0].Commitment, false)
	acc.Insert(blk.Transactions[0].GetCoinbaseTransaction().Outputs[1].Commitment, false)

	blk.Transactions[1].GetStakeTransaction().TxoRoot = acc.Root().Bytes()

	sigHash, err = blk.Transactions[1].GetStakeTransaction().SigHash()
	if err != nil {
		log.Fatal(err)
	}
	blk.Transactions[1].GetStakeTransaction().Signature, err = validatorKey.Sign(sigHash)
	if err != nil {
		log.Fatal(err)
	}

	merkleRoot := blockchain.TransactionsMerkleRoot(blk.Transactions)
	blk.Header.TxRoot = merkleRoot[:]

	out, err := json.MarshalIndent(blk, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}
