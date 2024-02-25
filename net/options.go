// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"time"
)

var ErrNetworkConfig = errors.New("network config error")

// Option is configuration option function for the Network
type Option func(cfg *config) error

func MempoolValidator(acceptToMempool func(tx *transactions.Transaction) error) Option {
	return func(cfg *config) error {
		cfg.acceptToMempool = acceptToMempool
		return nil
	}
}

func BlockValidator(validateBlock func(blk *blocks.XThinnerBlock, p peer.ID) error) Option {
	return func(cfg *config) error {
		cfg.validateBlock = validateBlock
		return nil
	}
}

func Params(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

func PrivateKey(privKey crypto.PrivKey) Option {
	return func(cfg *config) error {
		cfg.privateKey = privKey
		return nil
	}
}

func Datastore(ds repo.Datastore) Option {
	return func(cfg *config) error {
		cfg.datastore = ds
		return nil
	}
}

func MaxBanscore(max uint32) Option {
	return func(cfg *config) error {
		cfg.maxBanscore = max
		return nil
	}
}

func BanDuration(duration time.Duration) Option {
	return func(cfg *config) error {
		cfg.banDuration = duration
		return nil
	}
}

func UserAgent(s string) Option {
	return func(cfg *config) error {
		cfg.userAgent = s
		return nil
	}
}

func ListenAddrs(addrs []string) Option {
	return func(cfg *config) error {
		cfg.listenAddrs = addrs
		return nil
	}
}

func SeedAddrs(addrs []string) Option {
	return func(cfg *config) error {
		cfg.seedAddrs = addrs
		return nil
	}
}

func DisableNatPortMap() Option {
	return func(cfg *config) error {
		cfg.disableNatPortMap = true
		return nil
	}
}

func WithHost(host host.Host) Option {
	return func(cfg *config) error {
		cfg.host = host
		return nil
	}
}

func MaxMessageSize(maxMessageSize int) Option {
	return func(cfg *config) error {
		cfg.maxMessageSize = maxMessageSize
		return nil
	}
}

// ForceDHTServerMode forces the DHT to start in server mode.
// This is necessary if the node is a validator as they need
// to be publicly reachable.
func ForceDHTServerMode() Option {
	return func(cfg *config) error {
		cfg.forceServerMode = true
		return nil
	}
}

// TorBinary is the path to a tor binary file. If this option
// is used the tor transport is activated and the tor binary
// is started and managed by this process.
func TorBinary(binaryPath string) Option {
	return func(cfg *config) error {
		cfg.torBinary = binaryPath
		return nil
	}
}

// TorrcFile is the path to a custom tor torrc file. This allows
// you to set custom options for the running tor instance.
func TorrcFile(torrcPath string) Option {
	return func(cfg *config) error {
		cfg.torrcFile = torrcPath
		return nil
	}
}

// TorDualStack configures the network to make connections
// over both Tor AND the clear internet.
//
// Tor must be configured to use this mode.
func TorDualStack() Option {
	return func(cfg *config) error {
		cfg.torDualStack = true
		return nil
	}
}

// TorDataDir is the path to the directory to store
// tor data.
func TorDataDir(dir string) Option {
	return func(cfg *config) error {
		cfg.torDataDir = dir
		return nil
	}
}

type config struct {
	params            *params.NetworkParams
	userAgent         string
	seedAddrs         []string
	listenAddrs       []string
	disableNatPortMap bool
	maxMessageSize    int
	host              host.Host
	privateKey        crypto.PrivKey
	datastore         repo.Datastore
	acceptToMempool   func(tx *transactions.Transaction) error
	validateBlock     func(blk *blocks.XThinnerBlock, p peer.ID) error
	maxBanscore       uint32
	forceServerMode   bool
	banDuration       time.Duration
	torBinary         string
	torrcFile         string
	torDualStack      bool
	torDataDir        string
}

func (cfg *config) validate() error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrNetworkConfig)
	}
	if cfg.params == nil {
		return fmt.Errorf("%w: params is nil", ErrNetworkConfig)
	}
	if cfg.privateKey == nil && cfg.host == nil {
		return fmt.Errorf("%w: private key is nil", ErrNetworkConfig)
	}
	if cfg.listenAddrs == nil && cfg.host == nil {
		return fmt.Errorf("%w: listen addrs is nil", ErrNetworkConfig)
	}
	if cfg.datastore == nil && cfg.host == nil {
		return fmt.Errorf("%w: datastore is nil", ErrNetworkConfig)
	}
	if cfg.acceptToMempool == nil {
		return fmt.Errorf("%w: acceptToMempool is nil", ErrNetworkConfig)
	}
	if cfg.validateBlock == nil {
		return fmt.Errorf("%w: validateBlock is nil", ErrNetworkConfig)
	}
	if cfg.torDualStack && cfg.torBinary == "" {
		return fmt.Errorf("%w: dual stack mode requires tor binary path", ErrNetworkConfig)
	}
	return nil
}
