// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk"
	"time"
)

const (
	defaultSigCacheSize   = 100000
	defaultProofCacheSize = 100000
	defaultTransactionTTL = time.Hour * 24
)

// DefaultOptions returns a blockchain configure option that fills in
// the default settings. You will almost certainly want to override
// some of the defaults, such as parameters and datastore, etc.
func DefaultOptions() Option {
	return func(cfg *config) error {
		cfg.params = &params.RegestParams
		cfg.fpkb = repo.DefaultFeePerKilobyte
		cfg.minStake = repo.DefaultMinimumStake
		cfg.sigCache = blockchain.NewSigCache(defaultSigCacheSize)
		cfg.proofCache = blockchain.NewProofCache(defaultProofCacheSize)
		cfg.treasuryWhitelist = make(map[types.ID]bool)
		cfg.transactionTTL = defaultTransactionTTL
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

// Verifier is an implementation of the zk-snark Verifier interface.
//
// This option is required.
func Verifier(verifier zk.Verifier) Option {
	return func(cfg *config) error {
		cfg.verifier = verifier
		return nil
	}
}

// FeePerKilobyte is the minimum fee per byte to use when admitting
// transactions into the mempool. By extension the node will only
// relay transactions with a fee above this level as well.
func FeePerKilobyte(fpkb types.Amount) Option {
	return func(cfg *config) error {
		cfg.fpkb = fpkb
		return nil
	}
}

// MinStake is the minimum amount of stake that a stake transaction
// must post to be admitted into the mempool. By extension the node
// will only relay transactions with a stake above this level as well.
func MinStake(minStake types.Amount) Option {
	return func(cfg *config) error {
		cfg.minStake = minStake
		return nil
	}
}

// TreasuryWhitelist is a map of transactions ID that this node approves
// of for treasury withdrawls. Only this IDs will be accepted into the
// mempool.
func TreasuryWhitelist(whitelist []types.ID) Option {
	return func(cfg *config) error {
		m := make(map[types.ID]bool)
		for _, id := range whitelist {
			m[id] = true
		}
		cfg.treasuryWhitelist = m
		return nil
	}
}

// TransactionTTL represents the amount of time a transaction remains
// in the mempool without being included in a block before we discard it.
func TransactionTTL(ttl time.Duration) Option {
	return func(cfg *config) error {
		cfg.transactionTTL = ttl
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
	params            *params.NetworkParams
	chainView         ChainView
	fpkb              types.Amount
	minStake          types.Amount
	sigCache          *blockchain.SigCache
	proofCache        *blockchain.ProofCache
	verifier          zk.Verifier
	treasuryWhitelist map[types.ID]bool
	transactionTTL    time.Duration
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
	if cfg.verifier == nil {
		return AssertError("NewMempool: verifier cannot be nil")
	}
	return nil
}
