// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"github.com/ipfs/go-ds-badger"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/net"
	params "github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"go.uber.org/zap"
	"sort"
)

var log = zap.S()

// Server is the main class that brings all the constituent parts together
// into a full node.
type Server struct {
	cancelFunc context.CancelFunc
	config     *repo.Config
	params     *params.NetworkParams
	ds         repo.Datastore
	network    *net.Network
}

// BuildServer is the constructor for the server. We pass in the config file here
// and use it to configure all the various parts of the Server.
func BuildServer(config *repo.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Logging
	if err := setupLogging(config.LogDir, config.LogLevel, config.Testnet); err != nil {
		return nil, err
	}

	// Parameter selection
	var netParams *params.NetworkParams
	if config.Testnet {
		netParams = &params.Testnet1Params
	} else if config.Regest {
		netParams = &params.RegestParams
	} else {
		netParams = &params.MainnetParams
	}

	// Setup up badger datastore
	ds, err := badger.NewDatastore(config.DataDir, nil)
	if err != nil {
		return nil, err
	}

	// Load or create the private key for the node
	var privKey crypto.PrivKey
	has, err := repo.HasNetworkKey(ds)
	if err != nil {
		return nil, err
	}
	if has {
		privKey, err = repo.LoadNetworkKey(ds)
		if err != nil {
			return nil, err
		}
	} else {
		privKey, _, err = repo.GenerateNetworkKeypair()
		if err != nil {
			return nil, err
		}
		if err := repo.PutNetworkKey(ds, privKey); err != nil {
			return nil, err
		}
	}

	// Select seed addresses
	var seedAddrs []string
	if config.SeedAddrs != nil {
		seedAddrs = config.SeedAddrs
	} else {
		seedAddrs = netParams.SeedAddrs
	}

	// Select listen addresses
	var listenAddrs []string
	if config.ListenAddrs != nil {
		listenAddrs = config.ListenAddrs
	} else {
		listenAddrs = netParams.ListenAddrs
	}

	// Construct the Network.
	networkOpts := []net.Option{
		net.Datastore(ds),
		net.SeedAddrs(seedAddrs),
		net.ListenAddrs(listenAddrs),
		net.UserAgent(config.UserAgent),
		net.PrivateKey(privKey),
		net.Params(netParams),
	}
	if config.DisableNATPortMap {
		networkOpts = append(networkOpts, net.DisableNatPortMap())
	}

	network, err := net.NewNetwork(ctx, networkOpts...)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cancelFunc: cancel,
		config:     config,
		params:     netParams,
		ds:         ds,
		network:    network,
	}

	s.printListenAddrs()

	return s, nil
}

// Close shuts down all the parts of the server and blocks until
// they finish closing.
func (s *Server) Close() error {
	s.cancelFunc()
	if err := s.network.Close(); err != nil {
		return err
	}
	if err := s.ds.Close(); err != nil {
		return err
	}
	return nil
}

func (s *Server) printListenAddrs() {
	log.Infof("PeerID: %s", s.network.Host().ID().Pretty())
	var lisAddrs []string
	ifaceAddrs := s.network.Host().Addrs()
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Strings(lisAddrs)
	for _, addr := range lisAddrs {
		log.Infof("Listening on %s", addr)
	}
}
