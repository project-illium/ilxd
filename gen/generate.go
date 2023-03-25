// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"time"
)

const (
	BlockGenerationInterval = time.Second
	BlockVersion            = 1
)

type BlockGenerator struct {
	privKey        crypto.PrivKey
	ownPeerID      peer.ID
	ownPeerIDBytes []byte
	mpool          *mempool.Mempool
	blockchain     *blockchain.Blockchain
	tickInterval   time.Duration
	chain          *blockchain.Blockchain
	quit           chan struct{}
}

func NewBlockGenerator(opts ...Option) (*BlockGenerator, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	ownPeerID, err := peer.IDFromPrivateKey(cfg.privKey)
	if err != nil {
		return nil, err
	}
	ownPeerIDBytes, err := ownPeerID.Marshal()
	if err != nil {
		return nil, err
	}

	if cfg.tickInterval == time.Duration(0) {
		cfg.tickInterval = BlockGenerationInterval
	}

	g := &BlockGenerator{
		ownPeerID:      ownPeerID,
		ownPeerIDBytes: ownPeerIDBytes,
		privKey:        cfg.privKey,
		mpool:          cfg.mpool,
		tickInterval:   cfg.tickInterval,
		chain:          cfg.chain,
		quit:           make(chan struct{}),
	}

	go g.eventLoop()
	return g, nil
}

func (g *BlockGenerator) Close() {
	close(g.quit)
}

func (g *BlockGenerator) eventLoop() {
	ticker := time.NewTicker(g.tickInterval)
	for {
		select {
		case <-ticker.C:
			val := g.chain.WeightedRandomValidator()
			if val == g.ownPeerID {
				if err := g.generateBlock(); err != nil {
					log.Warnf("Error in block generator: %s", err.Error())
				}
			}

		case <-g.quit:
			return
		}
	}
}

func (g *BlockGenerator) generateBlock() error {
	bestID, height, timestamp := g.chain.BestBlock()

	now := time.Now()
	blockTime := now
	if !now.After(timestamp) {
		blockTime = timestamp.Add(time.Second)
	}
	// Don't generate a block if the timestamp would be too far into the future.
	if blockTime.After(now.Add(blockchain.MaxBlockFutureTime)) {
		return nil
	}

	blk := &blocks.Block{
		Header: &blocks.BlockHeader{
			Version:     1,
			Height:      height + 1,
			Parent:      bestID[:],
			Timestamp:   blockTime.Unix(),
			Producer_ID: g.ownPeerIDBytes,
		},
	}

	// The consensus rules prevent a stake tx and a spend of a staked
	// nullifier from being in the same block. We'll loop through
	// and remove any spends of stake if they were in the mempool.
	txs := g.mpool.GetTransactions()
	checkNullifiers := make(map[types.Nullifier]bool)
	for _, tx := range txs {
		if stake := tx.GetStakeTransaction(); stake != nil {
			checkNullifiers[types.NewNullifier(stake.Nullifier)] = true
		}
	}
	for txid, tx := range txs {
		switch t := tx.Tx.(type) {
		case *transactions.Transaction_StandardTransaction:
			for _, n := range t.StandardTransaction.Nullifiers {
				if checkNullifiers[types.NewNullifier(n)] {
					delete(txs, txid)
				}
			}
		case *transactions.Transaction_MintTransaction:
			for _, n := range t.MintTransaction.Nullifiers {
				if checkNullifiers[types.NewNullifier(n)] {
					delete(txs, txid)
				}
			}
		}
	}
	blk.Transactions = make([]*transactions.Transaction, 0, len(txs))
	for _, tx := range txs {
		blk.Transactions = append(blk.Transactions, tx)
	}

	merkles := blockchain.BuildMerkleTreeStore(blk.Transactions)
	blk.Header.TxRoot = merkles[len(merkles)-1]

	sigHash, err := blk.Header.SigHash()
	if err != nil {
		return err
	}
	sig, err := g.privKey.Sign(sigHash)
	if err != nil {
		return err
	}
	blk.Header.Signature = sig

	if err := g.blockchain.CheckConnectBlock(blk); err != nil {
		return err
	}

	// TODO: broadcast and send to consensus engine.
	return nil
}
