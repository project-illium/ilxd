// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types/transactions"
)

func DefaultOptions() Option {
	return func(cfg *config) error {
		networkPriv, _, err := repo.GenerateNetworkKeypair()
		if err != nil {
			return err
		}
		spendPriv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return err
		}
		cfg.networkKey = networkPriv
		cfg.spendKey = spendPriv
		cfg.params = &params.RegestParams
		cfg.datastore = mock.NewMapDatastore()
		cfg.nTxsPerBlock = 1
		cfg.initialCoins = (1 << 60) / 10
		return nil
	}
}

// Option is configuration option function for the Network
type Option func(cfg *config) error

func Params(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

func Datastore(ds repo.Datastore) Option {
	return func(cfg *config) error {
		cfg.datastore = ds
		return nil
	}
}

func NetworkKey(privKey crypto.PrivKey) Option {
	return func(cfg *config) error {
		cfg.networkKey = privKey
		return nil
	}
}

func SpendKey(privKey crypto.PrivKey) Option {
	return func(cfg *config) error {
		cfg.spendKey = privKey
		return nil
	}
}

func GenesisOutputs(outputs []*transactions.Output) Option {
	return func(cfg *config) error {
		cfg.genesisOutputs = outputs
		return nil
	}
}

func InitialCoins(n uint64) Option {
	return func(cfg *config) error {
		cfg.initialCoins = n
		return nil
	}
}

func NBlocks(n int) Option {
	return func(cfg *config) error {
		cfg.nBlocks = n
		return nil
	}
}

func NTxsPerBlock(n int) Option {
	return func(cfg *config) error {
		cfg.nTxsPerBlock = n
		return nil
	}
}

type config struct {
	params         *params.NetworkParams
	datastore      repo.Datastore
	networkKey     crypto.PrivKey
	spendKey       crypto.PrivKey
	genesisOutputs []*transactions.Output
	initialCoins   uint64
	nBlocks        int
	nTxsPerBlock   int
}

func (cfg *config) validate() error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.params == nil {
		return errors.New("params is nil")
	}
	if cfg.datastore == nil {
		return errors.New("datastore is nil")
	}
	if cfg.networkKey == nil {
		return errors.New("network key is nil")
	}
	if cfg.spendKey == nil {
		return errors.New("spend key is nil")
	}
	if cfg.initialCoins == 0 {
		return errors.New("initial coins is zero")
	}
	return nil
}
