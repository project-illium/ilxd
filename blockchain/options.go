// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/zk"
)

const (
	DefaultMaxTxoRoots    = 500
	DefaultMaxNullifiers  = 100000
	DefaultSigCacheSize   = 100000
	DefaultProofCacheSize = 100000
)

// DefaultOptions returns a blockchain configure option that fills in
// the default settings. You will almost certainly want to override
// some of the defaults, such as parameters and datastore, etc.
func DefaultOptions() Option {
	return func(cfg *config) error {
		cfg.params = &params.RegestParams
		cfg.datastore = mock.NewMockDatastore()
		cfg.sigCache = NewSigCache(DefaultSigCacheSize)
		cfg.proofCache = NewProofCache(DefaultProofCacheSize)
		cfg.maxNullifiers = DefaultMaxNullifiers
		cfg.maxTxoRoots = DefaultMaxTxoRoots
		return nil
	}
}

// Option is configuration option function for the blockchain
type Option func(cfg *config) error

// Params identifies which chain parameters the chain is associated
// with.
//
// This option is required.
func Params(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

// Datastore is an implementation of the repo.Datastore interface
//
// This option is required.
func Datastore(ds repo.Datastore) Option {
	return func(cfg *config) error {
		cfg.datastore = ds
		return nil
	}
}

// Verifier is an implementation of the zk.Verifier interface
// that is used to verify the zk-snark proofs.
//
// This option is required.
func Verifier(verifier zk.Verifier) Option {
	return func(cfg *config) error {
		cfg.verifier = verifier
		return nil
	}
}

// SignatureCache caches signature validation so we don't need to expend
// extra CPU to validate signatures more than once.
//
// If this is not provided a new instance will be used.
func SignatureCache(sigCache *SigCache) Option {
	return func(cfg *config) error {
		cfg.sigCache = sigCache
		return nil
	}
}

// SnarkProofCache caches proof validation so we don't need to expend
// extra CPU to validate zk-snark proofs more than once.
//
// If this is not provided a new instance will be used.
func SnarkProofCache(proofCache *ProofCache) Option {
	return func(cfg *config) error {
		cfg.proofCache = proofCache
		return nil
	}
}

// Indexer sets an IndexManager that is already configured with the desired
// indexers.
// These indexers will be notified whenever a new block is connected.
func Indexer(indexer IndexManager) Option {
	return func(cfg *config) error {
		cfg.indexManager = indexer
		return nil
	}
}

// MaxNullifiers is the maximum amount of nullifiers to hold in memory
// for fast access.
func MaxNullifiers(maxNullifiers uint) Option {
	return func(cfg *config) error {
		cfg.maxNullifiers = maxNullifiers
		return nil
	}
}

// MaxTxoRoots is the maximum amount of TxoRoots to hold in memory for
// fast access.
func MaxTxoRoots(maxTxoRoots uint) Option {
	return func(cfg *config) error {
		cfg.maxTxoRoots = maxTxoRoots
		return nil
	}
}

// Prune enables pruning of the blockchain. All historical blocks will be
// deleted from disk. This affects the ability to load these blocks from
// the API.
func Prune() Option {
	return func(cfg *config) error {
		cfg.prune = true
		return nil
	}
}

// Config specifies the blockchain configuration.
type config struct {
	params        *params.NetworkParams
	datastore     repo.Datastore
	sigCache      *SigCache
	proofCache    *ProofCache
	indexManager  IndexManager
	verifier      zk.Verifier
	maxNullifiers uint
	maxTxoRoots   uint
	prune         bool
}

func (cfg *config) validate() error {
	if cfg == nil {
		return AssertError("NewBlockchain: blockchain config cannot be nil")
	}
	if cfg.params == nil {
		return AssertError("NewBlockchain: params cannot be nil")
	}
	if cfg.datastore == nil {
		return AssertError("NewBlockchain: datastore cannot be nil")
	}
	if cfg.sigCache == nil {
		return AssertError("NewBlockchain: sig cache cannot be nil")
	}
	if cfg.proofCache == nil {
		return AssertError("NewBlockchain: proof cache cannot be nil")
	}
	if cfg.verifier == nil {
		return AssertError("NewBlockchain: verifier cannot be nil")
	}
	return nil
}
