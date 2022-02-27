// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/project-illium/ilxd/models/blocks"
	"path"
)

const (
	appProtocol = "/ilx"

	networkMainnet  = "mainnet"
	networkTestnet1 = "testnet1"
	networkRegtest  = "regtest"
)

type NetworkParams struct {
	ProtocolPrefix protocol.ID
	GenesisBlock   blocks.Block
	SeedAddrs      []string
	ListenAddrs    []string
	AddressPrefix  string
}

var MainnetParams = NetworkParams{
	ProtocolPrefix: protocol.ID(path.Join(appProtocol, networkMainnet)),
	GenesisBlock:   MainnetGenesisBlock,
	SeedAddrs: []string{
		"/ip4/167.172.126.176/tcp/4001/p2p/12D3KooWHnpVyu9XDeFoAVayqr9hvc9xPqSSHtCSFLEkKgcz5Wro",
	},
	ListenAddrs: []string{
		"/ip4/0.0.0.0/tcp/9001",
		"/ip6/::/tcp/9001",
		"/ip4/0.0.0.0/udp/9001/quic",
		"/ip6/::/udp/9001/quic",
	},
	AddressPrefix: "il",
}

var Testnet1Params = NetworkParams{
	ProtocolPrefix: protocol.ID(path.Join(appProtocol, networkTestnet1)),
	SeedAddrs: []string{
		"/ip4/167.172.126.176/tcp/4001/p2p/12D3KooWHnpVyu9XDeFoAVayqr9hvc9xPqSSHtCSFLEkKgcz5Wro",
	},
	ListenAddrs: []string{
		"/ip4/0.0.0.0/tcp/9002",
		"/ip6/::/tcp/9002",
		"/ip4/0.0.0.0/udp/9002/quic",
		"/ip6/::/udp/9002/quic",
	},
	AddressPrefix: "tn1",
}

var RegestParams = NetworkParams{
	ProtocolPrefix: protocol.ID(path.Join(appProtocol, networkRegtest)),
	ListenAddrs: []string{
		"/ip4/0.0.0.0/tcp/9003",
		"/ip6/::/tcp/9003",
		"/ip4/0.0.0.0/udp/9003/quic",
		"/ip6/::/udp/9003/quic",
	},
	AddressPrefix: "reg",
}
