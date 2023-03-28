// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/types/blocks"
	"time"
)

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
type AssertError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

// Option is configuration option function for the mempool
type Option func(cfg *config) error

// Blockchain is the node's active blockchain.
func Blockchain(chain *blockchain.Blockchain) Option {
	return func(cfg *config) error {
		cfg.chain = chain
		return nil
	}
}

// BroadcastFunc broadcasts the block to the network.
//
// This cannot be nil.
func BroadcastFunc(f func(blk *blocks.XThinnerBlock) error) Option {
	return func(cfg *config) error {
		cfg.broadcastFunc = f
		return nil
	}
}

// tickInterval is the interval that is used when determining
// whether or not we should be the one to create a new block.
// This is a consensus parameter and a node may be punished
// for creating blocks too quickly or too slowly.
// This is only an option here for testing.
func tickInterval(tickInterval time.Duration) Option {
	return func(cfg *config) error {
		cfg.tickInterval = tickInterval
		return nil
	}
}

// Mempool is the node's mempool object. This is needed
// to populate the block with transactions.
//
// This cannot be nil.
func Mempool(mpool *mempool.Mempool) Option {
	return func(cfg *config) error {
		cfg.mpool = mpool
		return nil
	}
}

// PrivateKey is the private key for the validator.
// It will be used to sign blocks.
//
// This cannot be nil
func PrivateKey(key crypto.PrivKey) Option {
	return func(cfg *config) error {
		cfg.privKey = key
		return nil
	}
}

// Config specifies the blockchain configuration.
type config struct {
	privKey       crypto.PrivKey
	mpool         *mempool.Mempool
	tickInterval  time.Duration
	chain         *blockchain.Blockchain
	broadcastFunc func(blk *blocks.XThinnerBlock) error
}

func (cfg *config) validate() error {
	if cfg == nil {
		return AssertError("NewBlockGenerator: mempool config cannot be nil")
	}
	if cfg.privKey == nil {
		return AssertError("NewBlockGenerator: private key cannot be nil")
	}
	if cfg.mpool == nil {
		return AssertError("NewBlockGenerator: mempool cannot be nil")
	}
	if cfg.chain == nil {
		return AssertError("NewBlockGenerator: WeightedChooser cannot be nil")
	}
	if cfg.broadcastFunc == nil {
		return AssertError("NewBlockGenerator: BroadcastFund cannot be nil")
	}
	return nil
}
