// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
)

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
type AssertError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

// Option is configuration option function for the blockchain
type Option func(cfg *config) error

// Params identifies which chain parameters the chain is associated
// with.
//
// This option is required.
func Params(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

// Network is the node's network implementation.
//
// This option is required.
func Network(n *net.Network) Option {
	return func(cfg *config) error {
		cfg.network = n
		return nil
	}
}

// Chooser is an implementation of the WeightedChooser used
// to select a validator to poll at random.
//
// This option is required.
func Chooser(chooser blockchain.WeightedChooser) Option {
	return func(cfg *config) error {
		cfg.chooser = chooser
		return nil
	}
}

// ValidatorConnector is an implementation of the ValidatorSetConnection interface
// used to determine if we have a validator set connections to provide consensus.
//
// This option is required.
func ValidatorConnector(valConn ValidatorSetConnection) Option {
	return func(cfg *config) error {
		cfg.valConn = valConn
		return nil
	}
}

// RequestBlock is a function which requests to download a block
// from the given peer.
//
// This option is required.
func RequestBlock(requestBlockFunc RequestBlockFunc) Option {
	return func(cfg *config) error {
		cfg.requestBlock = requestBlockFunc
		return nil
	}
}

// GetBlockID is a function which returns the blockID at the given height or
// an error.
//
// This option is required.
func GetBlockID(getBlockIDFunc GetBlockIDFunc) Option {
	return func(cfg *config) error {
		cfg.getBlockIDFunc = getBlockIDFunc
		return nil
	}
}

// PeerID is the node's own peerID.
//
// This option is required.
func PeerID(self peer.ID) Option {
	return func(cfg *config) error {
		cfg.self = self
		return nil
	}
}

// Config specifies the blockchain configuration.
type config struct {
	params         *params.NetworkParams
	network        *net.Network
	valConn        ValidatorSetConnection
	chooser        blockchain.WeightedChooser
	self           peer.ID
	requestBlock   RequestBlockFunc
	getBlockIDFunc GetBlockIDFunc
}

func (cfg *config) validate() error {
	if cfg == nil {
		return AssertError("NewConsensusEngine: config cannot be nil")
	}
	if cfg.params == nil {
		return AssertError("NewConsensusEngine: params cannot be nil")
	}
	if cfg.network == nil {
		return AssertError("NewConsensusEngine: network cannot be nil")
	}
	if cfg.valConn == nil {
		return AssertError("NewConsensusEngine: validator connector cannot be nil")
	}
	if cfg.chooser == nil {
		return AssertError("NewConsensusEngine: chooser cannot be nil")
	}
	if cfg.requestBlock == nil {
		return AssertError("NewConsensusEngine: requestBlockFunc cannot be nil")
	}
	if cfg.getBlockIDFunc == nil {
		return AssertError("NewConsensusEngine: getBlockIDFunc cannot be nil")
	}
	if cfg.self == "" {
		return AssertError("NewConsensusEngine: own peerID cannot be empty")
	}
	return nil
}
