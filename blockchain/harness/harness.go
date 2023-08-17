// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
)

type SpendableNote struct {
	Note            *types.SpendNote
	UnlockingScript *types.UnlockingScript
	PrivateKey      crypto.PrivKey
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
		if err := opt(&cfg); err != nil {
			return nil, err
		}
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

	genesis, spendableNote, err := createGenesisBlock(cfg.params, cfg.networkKey, cfg.spendKey, cfg.initialCoins, cfg.genesisOutputs)
	if err != nil {
		return nil, err
	}
	cfg.params.GenesisBlock = genesis
	for _, output := range genesis.Outputs() {
		harness.acc.Insert(output.Commitment, true)
	}

	commitment, err := spendableNote.Note.Commitment()
	if err != nil {
		return nil, err
	}
	proof, err := harness.acc.GetProof(commitment[:])
	if err != nil {
		return nil, err
	}

	nullifier := types.CalculateNullifier(proof.Index, spendableNote.Note.Salt, spendableNote.UnlockingScript.ScriptCommitment, spendableNote.UnlockingScript.ScriptParams...)
	harness.spendableNotes[nullifier] = spendableNote

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

func (h *TestHarness) GenerateBlockWithTransactions(txs []*transactions.Transaction, createdNotes []*SpendableNote) error {
	blk, err := h.generateBlockWithTransactions(txs)
	if err != nil {
		return err
	}
	if err := h.chain.ConnectBlock(blk, blockchain.BFNone); err != nil {
		return err
	}
	for _, out := range blk.Outputs() {
		h.acc.Insert(out.Commitment, true)
	}
	for _, sn := range createdNotes {
		commitment, err := sn.Note.Commitment()
		if err != nil {
			return err
		}
		proof, err := h.acc.GetProof(commitment[:])
		if err != nil {
			return err
		}
		nullifier := types.CalculateNullifier(proof.Index, sn.Note.Salt, sn.UnlockingScript.ScriptCommitment, sn.UnlockingScript.ScriptParams...)
		h.spendableNotes[nullifier] = sn
	}
	return nil
}

func (h *TestHarness) SpendableNotes() []*SpendableNote {
	notes := make([]*SpendableNote, 0, len(h.spendableNotes))
	for _, sn := range h.spendableNotes {
		notes = append(notes, sn)
	}
	return notes
}

func (h *TestHarness) Accumulator() *blockchain.Accumulator {
	return h.acc
}

func (h *TestHarness) Blockchain() *blockchain.Blockchain {
	return h.chain
}

func (h *TestHarness) Clone() (*TestHarness, error) {
	newHarness := &TestHarness{
		acc:            h.acc.Clone(),
		spendableNotes: make(map[types.Nullifier]*SpendableNote),
		validators:     make(map[peer.ID]*validator),
		txsPerBlock:    h.txsPerBlock,
		timeSource:     h.timeSource,
	}

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(h.chain.Params()))
	if err != nil {
		return nil, err
	}
	_, bestH, _ := h.chain.BestBlock()
	for i := uint32(1); i <= bestH; i++ {
		blk, err := h.chain.GetBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		err = chain.ConnectBlock(blk, blockchain.BFFastAdd)
		if err != nil {
			return nil, err
		}
	}
	newHarness.chain = chain

	for k, v := range h.spendableNotes {
		k2 := types.NewNullifier(make([]byte, len(k)))
		copy(k2[:], k[:])

		v2 := *v
		newHarness.spendableNotes[k2] = &v2
	}
	for k, v := range h.validators {
		k2 := k
		v2 := *v
		newHarness.validators[k2] = &v2
	}
	return newHarness, nil
}
