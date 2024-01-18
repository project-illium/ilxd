// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"math"
	"path"
)

const (
	appProtocol = "/ilx"

	networkMainnet  = "mainnet"
	networkTestnet1 = "testnet1"
	networkAlphanet = "alphanet"
	networkRegtest  = "regtest"
)

type Checkpoint struct {
	BlockID types.ID
	Height  uint32
}

type NetworkParams struct {
	// Name is a human-readable string to identify the params
	Name string

	// ProtocolPrefix defines the prefix for all network protocols.
	// Using different prefixes for different network effectively
	// segregates the networks as the handlers do not respond to
	// different protocol IDs.
	ProtocolPrefix protocol.ID

	// GenesisBlock defines the first block in the network. This
	// block must have a coinbase and stake transaction for the
	// network to move forward.
	GenesisBlock *blocks.Block

	// Checkpoints are known good blocks in the blockchain. We
	// use these to speed up the initial block download.
	Checkpoints []Checkpoint

	// SeedAddrs are used to connect to the network for the first
	// time. After first start up new peer addresses are stored in
	// the db and used to connect to the network.
	SeedAddrs []string

	// ListenAddrs defines the protocol, port, and interfaces that
	// the node will listen on. These are in multiaddr format.
	ListenAddrs []string

	// AddressPrefix defines the illium address prefix used as part
	// of the bech32 serialization.
	AddressPrefix string

	// The following controls the rate of coin emission for the network.
	//
	// EpochLength is the length of time (in seconds) between coinbase
	// distributions.
	EpochLength int64
	// TargetDistribution is the target number of coins to disperse
	// with an exponential decrease before the long term inflation rate
	// kicks in.
	TargetDistribution uint64
	// InitialDistributionPeriods defines the number of periods over which
	// the TargetDistribution will be emitted.
	InitialDistributionPeriods int64
	// AValue tweaks the first period's distribution according to the following:
	// w0 = ((TargetDistribution - GenesisCoinbase) / InitialDistributionPeriods) * AValue
	//
	// Since the distribution follows an exponential decay, the larger the first
	// period's distribution, the more coins will be distributed in InitialDistributionPeriods.
	// If you are changing the coin distribution parameters, you will want to
	// pick an AValue such that the total coins distributed over InitialDistributionPeriods
	// equals TargetDistribution - GenesisCoinbase.
	AValue float64
	// TreasuryPercentage is the percentage of new coins that are allocated
	// to the treasury.
	TreasuryPercentage float64
	// LongTermInflationRate defines the rate of emission per epoch after the
	// TargetDistribution is exhausted.
	LongTermInflationRate float64

	// AllowMockProofs sets whether the node be made to use mock proofs.
	// This is primarily for testing purposes as full proofs are very heavy.
	AllowMockProofs bool
}

var MainnetParams = NetworkParams{
	Name:           "mainnet",
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
	AddressPrefix:              "il",
	EpochLength:                60 * 60 * 24 * 7, // One week
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	AValue:                     2.59,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
	AllowMockProofs:            false,
}

var Testnet1Params = NetworkParams{
	Name:           "testnet1",
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
	AddressPrefix:              "tn1",
	EpochLength:                60 * 60 * 24 * 7, // One week
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	AValue:                     2.59,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
	AllowMockProofs:            false,
}

var AlphanetParams = NetworkParams{
	Name:           "alphanet",
	ProtocolPrefix: protocol.ID(path.Join(appProtocol, networkTestnet1)),
	SeedAddrs: []string{
		"/ip4/159.223.155.82/tcp/9002/p2p/12D3KooWKUMHDGvDuJjSkhey1Gz9kYPpt5Nw1wpzRtt9xwYWF1tx",
		"/ip4/137.184.35.103/tcp/9002/p2p/12D3KooWAqT761RNUN4ewfZwzCWkPDsG5BxMfbX48kdsT5qmjWLX",
	},
	ListenAddrs: []string{
		"/ip4/0.0.0.0/tcp/9002",
		"/ip6/::/tcp/9002",
		"/ip4/0.0.0.0/udp/9002/quic",
		"/ip6/::/udp/9002/quic",
	},
	AddressPrefix:              "al",
	GenesisBlock:               AlphanetGenesisBlock,
	EpochLength:                60 * 60 * 24 * 7, // One week
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	AValue:                     2.59,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
	AllowMockProofs:            false,
}

var RegestParams = NetworkParams{
	Name:           "regtest",
	ProtocolPrefix: protocol.ID(path.Join(appProtocol, networkRegtest)),
	ListenAddrs: []string{
		"/ip4/0.0.0.0/tcp/9003",
		"/ip6/::/tcp/9003",
		"/ip4/0.0.0.0/udp/9003/quic",
		"/ip6/::/udp/9003/quic",
	},
	SeedAddrs:                  []string{"/ip4/127.0.0.1/tcp/9003/p2p/12D3KooWN2RRWUokkcCjrf8zypvHwGv2u6rUepFAXheambSst5fV"},
	AddressPrefix:              "reg",
	GenesisBlock:               RegtestGenesisBlock,
	EpochLength:                60 * 3, // Three minutes
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	AValue:                     2.59,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
	AllowMockProofs:            true,
}
