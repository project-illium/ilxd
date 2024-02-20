// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import "github.com/libp2p/go-libp2p/core/peer"

type ValidatorSetConnection interface {
	ConnectedStakePercentage() float64
	RegisterDialSuccess(p peer.ID)
	RegisterDialFailure(p peer.ID)
}
