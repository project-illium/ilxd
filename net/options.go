// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
)

var ErrNetworkConfig = errors.New("network config error")

// Option is configuration option function for the Network
type Option func(cfg *config) error

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

type config struct {
	params            *params.NetworkParams
	userAgent         string
	seedAddrs         []string
	listenAddrs       []string
	disableNatPortMap bool
	host              host.Host
	privateKey        crypto.PrivKey
	datastore         repo.Datastore
}

func (cfg *config) validate() error {
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
	return nil
}
