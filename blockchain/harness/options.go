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
	"io"
	"os"
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
		cfg.params = &params.RegestParams
		cfg.datastore = mock.NewMapDatastore()
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
