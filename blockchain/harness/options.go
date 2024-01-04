// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"encoding/hex"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types/transactions"
)

const (
	defaultNetworkKey = "080112403dc9255cbaeb93921acf3589b8314433fc7ec0238c05d8559568ddf8da90b54a8d63fe6aaff827bef5bbfddcca441750ff617c439fb9866f9329b80afe22ada6"
	defaultSpendKey   = "080512407947fd46daa4ccf08db89e983c0d60f2571d6abdd1ffa595dc2adb8639f64905838e44ef2539cdd7f9ca9f32dbd0d59342afc2425e10c35b18d5ae2b4dcebd60"
)

func DefaultOptions() Option {
	return func(cfg *config) error {
		networkBytes, err := hex.DecodeString(defaultNetworkKey)
		if err != nil {
			return err
		}
		spendBytes, err := hex.DecodeString(defaultSpendKey)
		if err != nil {
			return err
		}
		networkPriv, err := crypto.UnmarshalPrivateKey(networkBytes)
		if err != nil {
			return err
		}
		spendPriv, err := crypto.UnmarshalPrivateKey(spendBytes)
		if err != nil {
			return err
		}
		cfg.networkKey = networkPriv
		cfg.spendKey = spendPriv
		cfg.pregenerate = 25000
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

func Pregenerate(nBlocks int) Option {
	return func(cfg *config) error {
		cfg.pregenerate = nBlocks
		return nil
	}
}

func Extension(extension bool) Option {
	return func(cfg *config) error {
		cfg.extension = true
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
	pregenerate    int
	extension      bool
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
