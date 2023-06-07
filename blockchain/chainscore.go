// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain/indexers"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"math"
	"sync"
)

type ChainScore float64

func (b *Blockchain) CalcChainScore(blks []*blocks.Block) (ChainScore, error) {
	b.stateLock.RLock()
	defer b.stateLock.RUnlock()

	tempds := mock.NewMapDatastore()
	tempChain := &Blockchain{
		params: b.params,
		ds:     tempds,
		index:  NewBlockIndex(tempds), // Reads and writes to mock in-memory db. Historical data not available.
		accumulatorDB: &AccumulatorDB{
			acc: b.accumulatorDB.Accumulator(),
			ds:  tempds,
			mtx: sync.RWMutex{},
		}, // Reads and writes to memory, disk writes are skipped by BFNoFlush,
		validatorSet: &ValidatorSet{
			params:       b.params,
			ds:           tempds,
			validators:   make(map[peer.ID]*Validator),
			nullifierMap: make(map[types.Nullifier]*Validator),
			toDelete:     make(map[peer.ID]struct{}),
			EpochBlocks:  b.validatorSet.EpochBlocks,
			mtx:          sync.RWMutex{},
		}, // Reads and writes to memory, disk writes are skipped by BFNoFlush.
		nullifierSet:      b.nullifierSet.Clone(), // Reads from both memory cache and disk db, writes to cache only.
		txoRootSet:        b.txoRootSet.Clone(),   // Reads from disk db, writes to cache only.
		sigCache:          NewSigCache(DefaultSigCacheSize),
		proofCache:        NewProofCache(DefaultProofCacheSize),
		indexManager:      indexers.NewIndexManager(nil, nil),
		scanner:           NewTransactionScanner(),
		notificationsLock: sync.RWMutex{},
		stateLock:         sync.RWMutex{},
	}
	defer tempChain.Close()

	for p, val := range b.validatorSet.validators {
		tempChain.validatorSet.validators[p] = val.Clone()
	}
	for n, val := range b.validatorSet.nullifierMap {
		tempChain.validatorSet.nullifierMap[n] = tempChain.validatorSet.validators[val.PeerID]
	}

	tempChain.txoRootSet.maxEntries = b.txoRootSet.maxEntries + uint(len(blks))

	tipHeader, err := b.index.Tip().Header()
	if err != nil {
		return 0, err
	}
	tempChain.index.ExtendIndex(tipHeader)

	balance, err := dsFetchTreasuryBalance(b.ds)
	if err != nil {
		return 0, err
	}
	if err := dsInitTreasury(tempChain.ds); err != nil {
		return 0, err
	}
	if err := dsInitCurrentSupply(tempChain.ds); err != nil {
		return 0, err
	}

	dbtx, err := tempChain.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return 0, err
	}
	defer dbtx.Discard(context.Background())

	if err := dsPutHeader(dbtx, tipHeader); err != nil {
		return 0, err
	}

	if err := dsCreditTreasury(dbtx, balance); err != nil {
		return 0, err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return 0, err
	}

	var (
		expectedBlocks = make(map[peer.ID]float64)
		actualBlocks   = make(map[peer.ID]float64)
	)
	for _, blk := range blks {
		if err := tempChain.ConnectBlock(blk, BFNoFlush); err != nil {
			return 0, err
		}

		for _, n := range blk.Nullifiers() {
			tempChain.nullifierSet.cachedEntries[n] = true
		}

		totalStaked := tempChain.validatorSet.TotalStaked()
		valID, err := peer.IDFromBytes(blk.Header.Producer_ID)
		if err != nil {
			return 0, err
		}
		for _, val := range tempChain.validatorSet.validators {
			expectedBlocks[val.PeerID] += float64(val.TotalStake) / float64(totalStaked)

			if val.PeerID == valID {
				actualBlocks[valID]++
			}
		}
	}

	score := ChainScore(0)
	for valID, expected := range expectedBlocks {
		actual, ok := actualBlocks[valID]
		if !ok {
			actual = 0
		}
		score += ChainScore(math.Abs(expected - actual))
	}

	return score, nil
}
