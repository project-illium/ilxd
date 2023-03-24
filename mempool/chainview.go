// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/libp2p/go-libp2p/core/peer"
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

	// UnclaimedCoins returns the number of unclaimed coins for a
	// given validator.
	UnclaimedCoins(validatorID peer.ID) (types.Amount, error)
}
