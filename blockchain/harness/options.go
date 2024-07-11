// Copyright (c) 2024 The illium developers
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
	"io"
	"os"
)

const (
	defaultNetworkKey = "08011240d483aa3529e1ddbd6e787a1693657ad0f8aaef6e00586b9843e03087a9800b23db9c4a2cb45dfd00321e8ec544585f4b2408552314daf8f0d4bdc2aa84c5a0e2"
	defaultSpendKey   = "080512400ebf107eff213ec000d5851409e3b593037fa3382894e2c8aab2bcb4d52fa3424d3f3f8862f0cf7a870182d8b8f87ded86d5844667eb531a070971fe0b6e4bab"
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
		cfg.params = &params.RegestParams
		cfg.datastore = mock.NewMockDatastore()
		cfg.nTxsPerBlock = 1
		cfg.initialCoins = (1 << 60) / 10
		return nil
	}
}

// Option is configuration option function for the Network
type Option func(cfg *config) error

// Params sets the parameters to initialize the chain with
func Params(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

// Datastore sets the harness' datastore
func Datastore(ds repo.Datastore) Option {
	return func(cfg *config) error {
		cfg.datastore = ds
		return nil
	}
}

// NetworkKey sets the harness' network key
func NetworkKey(privKey crypto.PrivKey) Option {
	return func(cfg *config) error {
		cfg.networkKey = privKey
		return nil
	}
}

// SpendKey sets the harness' spend key
func SpendKey(privKey crypto.PrivKey) Option {
	return func(cfg *config) error {
		cfg.spendKey = privKey
		return nil
	}
}

// LoadBlocks loads the requested number of blocks from the file
// instead of creating a new genesis block.
func LoadBlocks(f io.ReadCloser, nBlocks int) Option {
	return func(cfg *config) error {
		cfg.blockFiles = append(cfg.blockFiles, &blockFile{
			f:       f,
			nBlocks: nBlocks,
		})
		return nil
	}
}

// GenesisOutputs is any additional outputs to start the
// genesis block with.
func GenesisOutputs(outputs []*transactions.Output) Option {
	return func(cfg *config) error {
		cfg.genesisOutputs = outputs
		return nil
	}
}

// InitialCoins in the number of coins to start the chain with
func InitialCoins(n uint64) Option {
	return func(cfg *config) error {
		cfg.initialCoins = n
		return nil
	}
}

// NTxsPerBlock is the number of transactions to generate
// in each block.
func NTxsPerBlock(n int) Option {
	return func(cfg *config) error {
		cfg.nTxsPerBlock = n
		return nil
	}
}

// WriteToFile optionally write the harness blocks
// to a file.
func WriteToFile(f *os.File) Option {
	return func(cfg *config) error {
		cfg.writeToFile = f
		return nil
	}
}

type config struct {
	params         *params.NetworkParams
	datastore      repo.Datastore
	networkKey     crypto.PrivKey
	spendKey       crypto.PrivKey
	genesisOutputs []*transactions.Output
	writeToFile    *os.File
	blockFiles     []*blockFile
	initialCoins   uint64
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
