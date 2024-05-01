// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
)

// ChainView is an interface of methods that provide the blockchain
// context that the mempool needs to validate transactions.
type ChainView interface {
	// TreasuryBalance returns current balance of the treasury.
	TreasuryBalance() (types.Amount, error)

	//TxoRootExists returns whether the given txo root exists
	// in the txo root set.
	TxoRootExists(txoRoot types.ID) (bool, error)

	// NullifierExists returns whether the given nullifier exists
	// in the nullifier set.
	NullifierExists(n types.Nullifier) (bool, error)

	// GetValidator returns the validator for the given ID
	GetValidator(validatorID peer.ID) (*blockchain.Validator, error)

	// GetEpoch returns the last epoch ID of the blockchain
	GetEpoch() types.ID
}
