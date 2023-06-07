// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

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
	ValidatorKey            string `long:"validatorkey" description:"The genesis validator private key"`
	ViewKey                 string `long:"viewkey" description:"The genesis view public key"`
	InitialCoins            uint64 `long:"initialcoins" description:"The number of coins created by the genesis block"`
	CoinbaseUnlockingScript struct {
		SnarkVerificationKey string   `long:"unlockingscript.verificationkey" description:"The coinbase's 0th output's unlocking script verification key"`
		PublicParams         []string `long:"unlockingscript.params" description:"The coinbase's 0th output's unlocking script params"`
	}
	CoinbaseNote struct {
		ScriptHash string `long:"coinbasenote.scripthash" description:"The coinbase's 0th output's' note's' script hash"`
		Amount     uint64 `long:"coinbasenote.amount" description:"The coinbase's 0th output's' note's amount"`
		AssetID    string `long:"coinbasenote.assetid" description:"The coinbase's 0th output's' note's amount"`
		State      string `long:"coinbasenote.state" description:"The coinbase's 0th output's' note's state'"`
		Salt       string `long:"coinbasenote.salt" description:"The coinbase's 0th output's' note's salt'"`
	}
	AdditionalCoinbaseOutputs []transactions.Output
	Timestamp                 int64 `long:"timestamp" description:"The genesis block timestamp"`
}

// --validatorkey=080112401e08383f629522149a7505c7668f070bea3007af8ed2b3970e7a2f2456584039de64446ff3730ddbf2219fe1b996316650b747ea90d742c8e22f97bd3c0b616f
// --viewkey=08041220fd1264c92cc8d0b4fc8bdb811bbbc487734990c85cb42e333d6c86d95ce8b16b
// --initialcoins=230584300921369395
// --unlockingscript.verificationkey=0000000000000000000000000000000000000000000000000000000000000000
// --unlockingscript.params=0000000000000000000000000000000000000000000000000000000000000000
// --coinbasenote.scripthash=ae09db7cd54f42b490ef09b6bc541af688e4959bb8c53f359a6f56e38ab454a3
// --coinbasenote.amount=230584300921369395
// --coinbasenote.assetid=0000000000000000000000000000000000000000000000000000000000000000
// --coinbasenote.state=0000000000000000000000000000000000000000000000000000000000000000
// --coinbasenote.salt=09969db4b1ee03587e33c0e66a99fabbde33e3d10dd9642f9493cfac399956fd

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

	coinbaseNoteScriptHash, err := hex.DecodeString(params.CoinbaseNote.ScriptHash)
	if err != nil {
		log.Fatal(err)
	}
	coinbaseNoteAssetID, err := hex.DecodeString(params.CoinbaseNote.AssetID)
	if err != nil {
		log.Fatal(err)
	}
	coinbaseNoteState, err := hex.DecodeString(params.CoinbaseNote.State)
	if err != nil {
		log.Fatal(err)
	}
	coinbaseNoteSalt, err := hex.DecodeString(params.CoinbaseNote.Salt)
	if err != nil {
		log.Fatal(err)
	}

	note := types.SpendNote{
		ScriptHash: coinbaseNoteScriptHash,
		Amount:     types.Amount(params.CoinbaseNote.Amount),
		AssetID:    types.NewID(coinbaseNoteAssetID),
	}
	copy(note.State[:], coinbaseNoteState)
	copy(note.Salt[:], coinbaseNoteSalt)

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

	blk.Transactions[0] = transactions.WrapTransaction(&transactions.CoinbaseTransaction{
		Validator_ID: validatorIDBytes,
		NewCoins:     params.InitialCoins,
		Outputs:      make([]*transactions.Output, 1),
		Signature:    nil,
		Proof:        nil, // TODO
	})

	if types.NewID(note.ScriptHash).Compare(unlockingScript.Hash()) != 0 {
		log.Fatal("coinbase unlocking script does not match spend note script hash")
	}

	commitment, err := note.Commitment()
	if err != nil {
		log.Fatal(err)
	}

	cipherText, err := ilxcrypto.Encrypt(viewKey, note.Serialize())
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

	nullifier, err := types.CalculateNullifier(0, note.Salt, unlockingScript.SnarkVerificationKey, unlockingScript.PublicParams...)
	if err != nil {
		log.Fatal(err)
	}

	blk.Transactions[1] = transactions.WrapTransaction(&transactions.StakeTransaction{
		Validator_ID: validatorIDBytes,
		Amount:       params.CoinbaseNote.Amount,
		Nullifier:    nullifier[:],
		TxoRoot:      txoRoot,
		Locktime:     0,
		Signature:    nil,
		Proof:        nil, // TODO
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
	fmt.Println(string(out))
}
