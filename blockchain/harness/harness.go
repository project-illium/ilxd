// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
)

type SpendableNote struct {
	Note       *types.SpendNote
	PrivateKey crypto.PrivKey
}

type validator struct {
	networkKey crypto.PrivKey
}

type TestHarness struct {
	chain          *blockchain.Blockchain
	acc            *blockchain.Accumulator
	spendableNotes map[types.Nullifier]*SpendableNote
	validators     map[peer.ID]*validator
	txsPerBlock    int
	timeSource     int64
}

func NewTestHarness(opts ...Option) (*TestHarness, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	harness := &TestHarness{
		acc:            blockchain.NewAccumulator(),
		spendableNotes: make(map[types.Nullifier]*SpendableNote),
		validators:     make(map[peer.ID]*validator),
		txsPerBlock:    cfg.nTxsPerBlock,
	}

	genesis, spendNote, err := createGenesisBlock(cfg.params, cfg.networkKey, cfg.spendKey, cfg.initialCoins, cfg.genesisOutputs)
	if err != nil {
		return nil, err
	}
	cfg.params.GenesisBlock = *genesis
	for _, output := range genesis.Outputs() {
		harness.acc.Insert(output.Commitment, true)
	}

	commitment, err := spendNote.Commitment()
	if err != nil {
		return nil, err
	}
	proof, err := harness.acc.GetProof(commitment)
	if err != nil {
		return nil, err
	}

	nullifier, err := types.CalculateNullifier(proof.Index, spendNote.Salt, spendNote.UnlockingScript.SnarkVerificationKey, spendNote.UnlockingScript.PublicParams...)
	if err != nil {
		return nil, err
	}
	harness.spendableNotes[nullifier] = &SpendableNote{
		Note:       spendNote,
		PrivateKey: cfg.spendKey,
	}

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(cfg.params))
	if err != nil {
		return nil, err
	}
	harness.chain = chain

	validatorID, err := peer.IDFromPrivateKey(cfg.networkKey)
	if err != nil {
		return nil, err
	}
	harness.validators[validatorID] = &validator{
		networkKey: cfg.networkKey,
	}

	harness.timeSource = genesis.Header.Timestamp

	return harness, nil
}

func (h *TestHarness) GenerateBlocks(n int) error {
	blks, notes, err := h.generateBlocks(n)
	if err != nil {
		return err
	}
	for _, blk := range blks {
		if err := h.chain.ConnectBlock(blk, blockchain.BFNone); err != nil {
			return err
		}
		for _, out := range blk.Outputs() {
			h.acc.Insert(out.Commitment, true)
		}
	}
	h.spendableNotes = notes
	return nil
}
