// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/types"
)

type ChainView interface {
	TreasuryBalance() (uint64, error)

	TxoRootExists(txoRoot types.ID) (bool, error)

	NullifierExists(n types.Nullifier) (bool, error)

	UnclaimedCoins(validatorID peer.ID) (uint64, error)
}
