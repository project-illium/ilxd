// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"sort"
	"sync"
	"time"
)

const (
	// BlockGenerationInterval is the time to wait between generation attempts
	BlockGenerationInterval = time.Second

	// BlockVersion is the version to put in the block headers
	BlockVersion = 1

	// MinAllowableTimeBetweenDupBlocks is the minimum time the generator will
	// wait before creating another block at the same height. If the last block
	// got rejected we will want to eventually try again, but we don't want to
	// flood the network with blocks.
	MinAllowableTimeBetweenDupBlocks = time.Minute * 2

	serializedHeaderSize = 184

	// This is the number of extra bytes protobuf uses to
	// append a tx to the block. Not all transactions use
	// four bytes, but this is the maximum it would use if
	// the transaction was at the max size of 1MB.
	// If we want to be a little less lazy we could compute
	// the actual number per tx, but the extra bytes that
	// would fit under the soft limit would be tiny.
	txProtobufExtraBytes = 4
)

// BlockGenerator is used by validators to generate blocks.
// It strives to create blocks in proportion to the validator's
// share of the total weighted stake.
type BlockGenerator struct {
	privKey         crypto.PrivKey
	ownPeerID       peer.ID
	ownPeerIDBytes  []byte
	lastGenHeight   uint32
	lastGenTime     time.Time
	generatedBlocks map[types.ID]bool
	mpool           *mempool.Mempool
	policy          *policy.Policy
	tickInterval    time.Duration
	chain           *blockchain.Blockchain
	broadcast       func(blk *blocks.XThinnerBlock) error
	active          bool
	activeMtx       sync.RWMutex
	genBlksMtx      sync.RWMutex
	lastHeight      uint32
	interruptChan   chan uint32
	quit            chan struct{}
}

// NewBlockGenerator returns a new BlockGenerator but does not
// start it. Start needs to be called separately.
func NewBlockGenerator(opts ...Option) (*BlockGenerator, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
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
		ownPeerID:       ownPeerID,
		ownPeerIDBytes:  ownPeerIDBytes,
		privKey:         cfg.privKey,
		generatedBlocks: make(map[types.ID]bool),
		mpool:           cfg.mpool,
		tickInterval:    cfg.tickInterval,
		chain:           cfg.chain,
		policy:          cfg.policy,
		broadcast:       cfg.broadcastFunc,

		activeMtx:     sync.RWMutex{},
		genBlksMtx:    sync.RWMutex{},
		interruptChan: make(chan uint32),
		active:        false,
	}

	return g, nil
}

// Start turns on the generator and starts generating blocks
func (g *BlockGenerator) Start() {
	g.activeMtx.Lock()
	defer g.activeMtx.Unlock()
	if g.active {
		return
	}
	g.active = true

	g.quit = make(chan struct{})
	go g.eventLoop()
	log.Info("Block generator active")
}

// Close shuts down the generator and puts it into a state
// where it can be restarted if necessary.
func (g *BlockGenerator) Close() {
	g.activeMtx.Lock()
	defer g.activeMtx.Unlock()

out:
	for {
		select {
		case <-g.interruptChan:
		default:
			break out
		}
	}

	if g.active {
		g.active = false
		close(g.quit)
	}
}

// Active returns whether the block generator is currently active
func (g *BlockGenerator) Active() bool {
	g.activeMtx.RLock()
	defer g.activeMtx.RUnlock()

	return g.active
}

// Interrupt should be called when a new block is connected to the
// chain. It will stop the generator goroutine from creating a
// block at the current height and move on to the next height.
func (g *BlockGenerator) Interrupt(height uint32) {
	go func() {
		if g.Active() {
			g.interruptChan <- height
		}
	}()
}

// IsOwnBlock returns whether the block was created by the generator
func (g *BlockGenerator) IsOwnBlock(blkID types.ID) bool {
	g.genBlksMtx.RLock()
	defer g.genBlksMtx.RUnlock()
	return g.generatedBlocks[blkID]
}

func (g *BlockGenerator) eventLoop() {
	ticker := time.NewTicker(g.tickInterval)
	for {
		select {
		case <-ticker.C:
			val := g.chain.WeightedRandomValidator()
			if val == g.ownPeerID {
				if err := g.generateBlock(); err != nil {
					log.WithCaller(true).Error("Error in block generator", log.Args("error", err))
				}
			}
		case <-g.quit:
			return
		}
	}
}

func (g *BlockGenerator) generateBlock() error {
	underlimit, current, max, err := g.chain.IsProducerUnderLimit(g.ownPeerID)
	if err != nil {
		return err
	}

	now := time.Now()
	bestID, height, timestamp := g.chain.BestBlock()
	lastHeight := g.lastHeight
	g.lastHeight = height
	if !underlimit {
		if height > lastHeight {
			log.Warn("Block generator over limit", log.ArgsFromMap(map[string]any{
				"epoch blocks": current,
				"limit":        max,
			}))
		}
		return nil
	}

	if g.lastGenHeight != 0 && g.lastGenHeight == height+1 && now.Before(g.lastGenTime.Add(MinAllowableTimeBetweenDupBlocks)) {
		return nil
	}

	blockTime := now.Unix()
	if blockTime <= timestamp.Unix() {
		blockTime = timestamp.Unix() + 1
	}
	// Don't generate a block if the timestamp would be too far into the future.
	if blockTime > timestamp.Unix()+int64(blockchain.MaxBlockFutureTime) {
		return nil
	}

	blk := &blocks.Block{
		Header: &blocks.BlockHeader{
			Version:     BlockVersion,
			Height:      height + 1,
			Parent:      bestID[:],
			Timestamp:   blockTime,
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

	blockSize := serializedHeaderSize
	for txid, tx := range txs {
		switch t := tx.Tx.(type) {
		case *transactions.Transaction_StandardTransaction:
			for _, n := range t.StandardTransaction.Nullifiers {
				if checkNullifiers[types.NewNullifier(n)] {
					delete(txs, txid)
				}
			}
			if (t.StandardTransaction.Locktime != nil && t.StandardTransaction.Locktime.Timestamp > 0) &&
				(t.StandardTransaction.Locktime.Timestamp > blockTime+t.StandardTransaction.Locktime.Precision ||
					t.StandardTransaction.Locktime.Timestamp < blockTime-t.StandardTransaction.Locktime.Precision) {
				delete(txs, txid)
			}
		case *transactions.Transaction_MintTransaction:
			for _, n := range t.MintTransaction.Nullifiers {
				if checkNullifiers[types.NewNullifier(n)] {
					delete(txs, txid)
				}
			}

			if (t.MintTransaction.Locktime != nil && t.MintTransaction.Locktime.Timestamp > 0) &&
				(t.MintTransaction.Locktime.Timestamp > blockTime+t.MintTransaction.Locktime.Precision ||
					t.MintTransaction.Locktime.Timestamp < blockTime-t.MintTransaction.Locktime.Precision) {
				delete(txs, txid)
			}
		}
	}
	if len(txs) == 0 {
		return nil
	}
	blk.Transactions = make([]*transactions.Transaction, 0, len(txs))
	for _, tx := range txs {
		txSize, err := tx.SerializedSize()
		if err != nil {
			return err
		}
		blockSize += txSize + txProtobufExtraBytes

		if uint32(blockSize) > g.policy.GetBlocksizeSoftLimit() {
			break
		}
		blk.Transactions = append(blk.Transactions, tx)
	}

	sort.Sort(mempool.TxSorter(blk.Transactions))

	merkleRoot := blockchain.TransactionsMerkleRoot(blk.Transactions)
	blk.Header.TxRoot = merkleRoot[:]

	sigHash, err := blk.Header.SigHash()
	if err != nil {
		return err
	}
	sig, err := g.privKey.Sign(sigHash)
	if err != nil {
		return err
	}
	blk.Header.Signature = sig

	if invalidIdx, err := g.chain.CheckConnectBlock(blk); err != nil {
		if invalidIdx >= 0 && invalidIdx < len(blk.Transactions) {
			invalidTx := blk.Transactions[invalidIdx]
			g.mpool.RemoveBlockTransactions([]*transactions.Transaction{invalidTx})
		}
		return err
	}

	xthinnerBlock, err := g.mpool.EncodeXthinner(blk.Txids())
	if err != nil {
		return err
	}
	xthinnerBlock.Header = blk.Header

out:
	for {
		select {
		case height := <-g.interruptChan:
			if height >= blk.Header.Height {
				return nil
			}
		default:
			break out
		}
	}

	g.lastGenHeight = blk.Header.Height
	g.lastGenTime = time.Unix(blk.Header.Timestamp, 0)

	log.Debug("Generated block", log.ArgsFromMap(map[string]any{
		"id":     blk.ID().String(),
		"height": blk.Header.Height,
	}))

	blkID := blk.ID()
	g.genBlksMtx.Lock()
	g.generatedBlocks[blkID] = true
	g.genBlksMtx.Unlock()

	time.AfterFunc(time.Minute*5, func() {
		g.genBlksMtx.Lock()
		delete(g.generatedBlocks, blkID)
		g.genBlksMtx.Unlock()
	})

	return g.broadcast(xthinnerBlock)
}
