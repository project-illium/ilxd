// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
)

const (
	defaultFeePerByte     = 500
	defaultMinimumStake   = 1000000
	defaultSigCacheSize   = 100000
	defaultProofCacheSize = 100000
)

// DefaultOptions returns a blockchain configure option that fills in
// the default settings. You will almost certainly want to override
// some of the defaults, such as parameters and datastore, etc.
func DefaultOptions() Option {
	return func(cfg *config) error {
		cfg.params = &params.RegestParams
		cfg.fpb = defaultFeePerByte
		cfg.minStake = defaultMinimumStake
		cfg.sigCache = blockchain.NewSigCache(defaultSigCacheSize)
		cfg.proofCache = blockchain.NewProofCache(defaultProofCacheSize)
		return nil
	}
}

// Option is configuration option function for the mempool
type Option func(cfg *config) error

// BlockchainView is an interface that is used to access
// data from the blockchain needed for mempool validation.
//
// This option is required.
func BlockchainView(cv ChainView) Option {
	return func(cfg *config) error {
		cfg.chainView = cv
		return nil
	}
}

// Params identifies which chain parameters the mempool is associated
// with.
//
// This option is required.
func Params(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

// FeePerByte is the minimum fee per byte to use when admitting
// transactions into the mempool. By extension the node will only
// relay transactions with a fee above this level as well.
func FeePerByte(fpb uint64) Option {
	return func(cfg *config) error {
		cfg.fpb = fpb
		return nil
	}
}

// MinStake is the minimum amount of stake that a stake transaction
// must post to be admitted into the mempool. By extension the node
// will only relay transactions with a stake above this level as well.
func MinStake(minStake uint64) Option {
	return func(cfg *config) error {
		cfg.minStake = minStake
		return nil
	}
}

// SignatureCache caches signature validation so we don't need to expend
// extra CPU to validate signatures more than once.
//
// If this is not provided a new instance will be used.
func SignatureCache(sigCache *blockchain.SigCache) Option {
	return func(cfg *config) error {
		cfg.sigCache = sigCache
		return nil
	}
}

// ProofCache caches proof validation so we don't need to expend
// extra CPU to validate zero knowledge proofs more than once.
//
// If this is not provided a new instance will be used.
func ProofCache(proofCache *blockchain.ProofCache) Option {
	return func(cfg *config) error {
		cfg.proofCache = proofCache
		return nil
	}
}

// Config specifies the blockchain configuration.
type config struct {
	params     *params.NetworkParams
	chainView  ChainView
	fpb        uint64
	minStake   uint64
	sigCache   *blockchain.SigCache
	proofCache *blockchain.ProofCache
}

func (cfg *config) validate() error {
	if cfg == nil {
		return AssertError("NewMempool: mempool config cannot be nil")
	}
	if cfg.chainView == nil {
		return AssertError("NewMempool: chainview cannot be nil")
	}
	if cfg.params == nil {
		return AssertError("NewMempool: params cannot be nil")
	}
	if cfg.sigCache == nil {
		return AssertError("NewMempool: sig cache cannot be nil")
	}
	if cfg.proofCache == nil {
		return AssertError("NewMempool: proof cache cannot be nil")
	}
	return nil
}
