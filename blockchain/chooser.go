// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

// WeightedChooser is an interface for the Blockchain's
// WeightedRandomValidator method.
type WeightedChooser interface {
	WeightedRandomValidator() peer.ID
}
