// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/wallet"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"time"
)

type TestHarness struct {
	chain *blockchain.Blockchain
}

func NewTestHarness(opts ...Option) (*TestHarness, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	genesis, _, err := createGenesisBlock(cfg.params, cfg.networkKey, cfg.spendKey, cfg.initialCoins, cfg.genesisOutputs)
	if err != nil {
		return nil, err
	}
	cfg.params.GenesisBlock = *genesis

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(cfg.params))
	if err != nil {
		return nil, err
	}

	return &TestHarness{
		chain: chain,
	}, nil
}

func createGenesisBlock(params *params.NetworkParams, networkKey, spendKey crypto.PrivKey,
	initialCoins uint64, additionalOutputs []*transactions.Output) (*blocks.Block, *wallet.SpendNote, error) {

	// First we'll create the spend note for the coinbase transaction.
	// The initial coins will be generated to the spendKey.
	var salt [32]byte
	rand.Read(salt[:])

	note := &wallet.SpendNote{
		SpendScript: wallet.SpendScript{
			Threshold: 1,
			Pubkeys:   []*wallet.TimeLockedPubkey{wallet.NewTimeLockedPubkey(spendKey.GetPublic(), time.Time{})},
		},
		Amount:  initialCoins,
		AssetID: wallet.IlliumCoinID,
		Salt:    salt,
	}

	// Next we're going to start building the coinbase transaction
	commitment, err := note.Commitment()
	if err != nil {
		return nil, nil, err
	}
	validatorID, err := peer.IDFromPublicKey(networkKey.GetPublic())
	if err != nil {
		return nil, nil, err
	}
	idBytes, err := validatorID.Marshal()
	if err != nil {
		return nil, nil, err
	}

	coinbaseTx := &transactions.CoinbaseTransaction{
		Validator_ID: idBytes,
		NewCoins:     initialCoins,
		Outputs: []*transactions.Output{
			{
				Commitment:      commitment,
				EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
				Ciphertext:      make([]byte, blockchain.CipherTextLen),
			},
		},
	}
	coinbaseTx.Outputs = append(coinbaseTx.Outputs, additionalOutputs...)

	// And now sign the coinbase transaction with the network key
	sigHash, err := coinbaseTx.SigHash()
	if err != nil {
		return nil, nil, err
	}

	sig, err := networkKey.Sign(sigHash)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx.Signature = sig

	// Finally we're going to create the zk-snark proof for the coinbase
	// transaction.
	spendScriptHash, err := note.SpendScript.Hash()
	if err != nil {
		return nil, nil, err
	}

	serializedPubkey, err := note.SpendScript.Pubkeys[0].Serialize()
	if err != nil {
		return nil, nil, err
	}

	nullifier, err := types.CalculateNullifier(0, salt, 1, serializedPubkey)
	if err != nil {
		return nil, nil, err
	}

	publicParams := &standard.PublicParams{
		OutputCommitments: [][]byte{commitment},
		Nullifiers:        [][]byte{nullifier.Bytes()},
		Fee:               0,
		Coinbase:          initialCoins,
	}
	privateParams := &standard.PrivateParams{
		Outputs: []standard.PrivateOutput{
			{
				SpendScript: spendScriptHash,
				Amount:      initialCoins,
				Salt:        note.Salt[:],
				AssetID:     note.AssetID,
			},
		},
	}

	proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx.Proof = proof

	// Next we have to build the transaction staking the coins generated
	// in the prior coinbase transaction. This is needed because if no
	// validators are set in the genesis block we can't move the chain
	// forward.
	//
	// Notice there is a special validation rule for the genesis block
	// that doesn't apply to any other block. Normally, transactions
	// must contain a txoRoot for a block already in the chain. However,
	// in the case of the genesis block there are no other blocks in the
	// chain yet. So the rules allow the genesis block to reference its
	// own txoRoot.
	acc := blockchain.NewAccumulator()
	for i, output := range coinbaseTx.Outputs {
		acc.Insert(output.Commitment, i == 0)
	}
	txoRoot := acc.Root()
	inclusionProof, err := acc.GetProof(commitment)
	if err != nil {
		return nil, nil, err
	}

	stakeTx := &transactions.StakeTransaction{
		Validator_ID: idBytes,
		Amount:       initialCoins,
		Nullifier:    nullifier.Bytes(),
		TxoRoot:      txoRoot.Bytes(), // See note above
	}

	// Sign the stake transaction
	sigHash2, err := stakeTx.SigHash()
	if err != nil {
		return nil, nil, err
	}

	sig2, err := networkKey.Sign(sigHash2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Signature = sig2

	// And generate the zk-snark proof
	sig3, err := spendKey.Sign(sigHash2)
	if err != nil {
		return nil, nil, err
	}

	publicParams2 := &stake.PublicParams{
		TXORoot:   txoRoot.Bytes(),
		SigHash:   sigHash2,
		Amount:    initialCoins,
		Nullifier: nullifier.Bytes(),
	}
	privateParams2 := &stake.PrivateParams{
		AssetID:         wallet.IlliumCoinID[:],
		Salt:            salt[:],
		CommitmentIndex: 0,
		InclusionProof: standard.InclusionProof{
			Hashes:      inclusionProof.Hashes,
			Flags:       inclusionProof.Flags,
			Accumulator: inclusionProof.Accumulator,
		},
		Threshold:   1,
		Pubkeys:     [][]byte{serializedPubkey},
		Signatures:  [][]byte{sig3},
		SigBitfield: 1,
	}

	proof2, err := zk.CreateSnark(stake.StakeCircuit, privateParams2, publicParams2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Proof = proof2

	// Now we add the transactions to the genesis block
	genesis := params.GenesisBlock
	genesis.Transactions = []*transactions.Transaction{
		{Tx: &transactions.Transaction_CoinbaseTransaction{CoinbaseTransaction: coinbaseTx}},
		{Tx: &transactions.Transaction_StakeTransaction{StakeTransaction: stakeTx}},
	}

	// And create the genesis merkle root
	txids := make([][]byte, 0, len(genesis.Transactions))
	for _, tx := range genesis.Transactions {
		txids = append(txids, tx.ID().Bytes())
	}
	merkles := blockchain.BuildMerkleTreeStore(txids)
	genesis.Header.TxRoot = merkles[len(merkles)-1]
	genesis.Header.Timestamp = time.Now().Unix()

	return &genesis, note, nil
}
