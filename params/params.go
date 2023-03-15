// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/project-illium/ilxd/types/blocks"
	"math"
	"path"
)

const (
	appProtocol = "/ilx"

	networkMainnet  = "mainnet"
	networkTestnet1 = "testnet1"
	networkRegtest  = "regtest"
)

type NetworkParams struct {
	// ProtocolPrefix defines the prefix for all network protocols.
	// Using different prefixes for different network effectively
	// segregates the networks as the handlers do not respond to
	// different protocol IDs.
	ProtocolPrefix protocol.ID

	// GenesisBlock defines the first block in the network. This
	// block must have a coinbase and stake transaction for the
	// network to move forward.
	GenesisBlock *blocks.Block

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
	// DecayFactor the rate of decay for the above mentioned exponential
	// decrease in emission. This controls not just the rate of decrease
	// but how long it takes to emit the TargetDistribution.
	DecayFactor float64
	// InitialDistributionPeriods defines the number of periods over which
	// the TargetDistribution will be emitted.
	InitialDistributionPeriods int64
	// RValue is used in the coin emission calculation.
	// TargetDistribution * LongTermInflationRate * r is added to the epoch's
	// distribution. This can be tweaked to target a specific distribution in
	// the final epoch of the initial distribution.
	RValue float64
	// AValue is the first epoch's distribution. This value is decreased exponentially
	// in subsequent epochs.
	AValue float64
	// TreasuryPercentage is the percentage of new coins that are allocated
	// to the treasury.
	TreasuryPercentage float64
	// LongTermInflationRate defines the rate of emission per epoch after the
	// TargetDistribution is exhausted.
	LongTermInflationRate float64
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
	AddressPrefix:              "il",
	EpochLength:                60 * 60 * 24 * 7, // One week
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	DecayFactor:                .0050102575551808,
	RValue:                     .23,
	AValue:                     4724093425051351,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
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
	AddressPrefix:              "tn1",
	EpochLength:                60 * 60 * 24 * 7, // One week
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	DecayFactor:                .0050102575551808,
	RValue:                     .23,
	AValue:                     4724093425051351,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
}

var RegestParams = NetworkParams{
	ProtocolPrefix: protocol.ID(path.Join(appProtocol, networkRegtest)),
	ListenAddrs: []string{
		"/ip4/0.0.0.0/tcp/9003",
		"/ip6/::/tcp/9003",
		"/ip4/0.0.0.0/udp/9003/quic",
		"/ip6/::/udp/9003/quic",
	},
	AddressPrefix:              "reg",
	GenesisBlock:               RegtestGenesisBlock,
	EpochLength:                60 * 60 * 24 * 7, // One week
	TargetDistribution:         1 << 60,
	InitialDistributionPeriods: 520,
	DecayFactor:                .0050102575551808,
	RValue:                     .23,
	AValue:                     4724093425051351,
	TreasuryPercentage:         5,
	LongTermInflationRate:      math.Pow(1.02, 1.0/52) - 1, // Annualizes to 2% over 52 periods.
}
