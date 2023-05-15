// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	ilxcrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"log"
)

type GenerationParams struct {
	ValidatorKey            string
	ViewKey                 string
	InitialCoins            uint64
	CoinbaseUnlockingScript struct {
		SnarkVerificationKey string   `json:"snarkVerificationKey"`
		PublicParams         []string `json:"publicParams"`
	}
	CoinbaseNote struct {
		ScriptHash string `json:"scriptHash"`
		Amount     uint64 `json:"amount"`
		AssetID    string `json:"assetID"`
		State      string `json:"state"`
		Salt       string `json:"salt"`
	}
	AdditionalCoinbaseOutputs []transactions.Output
	Timestamp                 int64
}

func main() {
	var params GenerationParams
	parser := flags.NewNamedParser("genesis generator", flags.Default)
	parser.AddGroup("Generation Options", "Options for generating a new genesis block", &params)
	if _, err := parser.Parse(); err != nil {
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
		Transactions: make([]*transactions.Transaction, 2),
	}

	validatorKeyBytes, err := hex.DecodeString(params.ValidatorKey)
	if err != nil {
		log.Fatal(err)
	}

	validatorKey, err := crypto.UnmarshalPrivateKey(validatorKeyBytes)
	if err != nil {
		log.Fatal(err)
	}

	viewKeyBytes, err := hex.DecodeString(params.ViewKey)
	if err != nil {
		log.Fatal(err)
	}

	viewKey, err := crypto.UnmarshalPublicKey(viewKeyBytes)
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

	snarkKeyBytes, err := hex.DecodeString(params.CoinbaseUnlockingScript.SnarkVerificationKey)
	if err != nil {
		log.Fatal(err)
	}

	unlockingParams := make([][]byte, 0, len(params.CoinbaseUnlockingScript.PublicParams))
	for _, p := range params.CoinbaseUnlockingScript.PublicParams {
		params, err := hex.DecodeString(p)
		if err != nil {
			log.Fatal(err)
		}
		unlockingParams = append(unlockingParams, params)
	}

	unlockingScript := types.UnlockingScript{
		SnarkVerificationKey: snarkKeyBytes,
		PublicParams:         unlockingParams,
	}

	blk.Transactions[0] = transactions.WrapTransaction(&transactions.Transaction_CoinbaseTransaction{
		CoinbaseTransaction: &transactions.CoinbaseTransaction{
			Validator_ID: validatorIDBytes,
			NewCoins:     params.InitialCoins,
			Outputs:      make([]*transactions.Output, 1),
			Signature:    nil,
			Proof:        nil, // TODO
		},
	})

	if types.NewID(params.CoinbaseNote.ScriptHash).Compare(unlockingScript.Hash()) != 0 {
		log.Fatal("coinbase unlocking script does not match spend note script hash")
	}

	commitment, err := params.CoinbaseNote.Commitment()
	if err != nil {
		log.Fatal(err)
	}

	cipherText, err := ilxcrypto.Encrypt(viewKey, params.CoinbaseNote.Serialize())
	if err != nil {
		log.Fatal(err)
	}

	output0 := &transactions.Output{
		Commitment: commitment,
		Ciphertext: cipherText,
	}
	blk.Transactions[0].GetCoinbaseTransaction().Outputs[0] = output0

	for _, out := range params.AdditionalCoinbaseOutputs {
		blk.Transactions[0].GetMintTransaction().Outputs = append(blk.Transactions[0].GetMintTransaction().Outputs, &out)
	}

	sigHash, err := blk.Transactions[0].GetCoinbaseTransaction().SigHash()
	if err != nil {
		log.Fatal(err)
	}
	sig, err := validatorKey.Sign(sigHash)
	if err != nil {
		log.Fatal(err)
	}
	blk.Transactions[0].GetCoinbaseTransaction().Signature = sig

	acc := blockchain.NewAccumulator()
	for _, output := range blk.Transactions[0].GetCoinbaseTransaction().Outputs {
		acc.Insert(output.Commitment, false)
	}
	txoRoot := acc.Root().Bytes()

	nullifier, err := types.CalculateNullifier(0, params.CoinbaseNote.Salt, unlockingScript.SnarkVerificationKey, unlockingScript.PublicParams...)
	if err != nil {
		log.Fatal(err)
	}

	blk.Transactions[1] = transactions.WrapTransaction(&transactions.Transaction_StakeTransaction{
		StakeTransaction: &transactions.StakeTransaction{
			Validator_ID: validatorIDBytes,
			Amount:       params.CoinbaseNote.Amount,
			Nullifier:    nullifier[:],
			TxoRoot:      txoRoot,
			Locktime:     0,
			Signature:    nil,
			Proof:        nil, // TODO
		},
	})

	sigHash, err = blk.Transactions[1].GetStakeTransaction().SigHash()
	if err != nil {
		log.Fatal(err)
	}
	sig, err = validatorKey.Sign(sigHash)
	if err != nil {
		log.Fatal(err)
	}
	blk.Transactions[1].GetStakeTransaction().Signature = sig

	merkles := blockchain.BuildMerkleTreeStore(blk.Transactions)
	blk.Header.TxRoot = merkles[len(merkles)-1]

	out, err := json.MarshalIndent(blk, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
