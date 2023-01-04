// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/net"
	"math/rand"
)

type WeightedChooser interface {
	WeightedRandomPeer() peer.ID
}

type MockChooser struct {
	network *net.Network
}

func (m *MockChooser) WeightedRandomPeer() peer.ID {
	peers := m.network.Host().Network().Peers()
	l := len(peers)
	if l == 0 {
		return ""
	}
	i := rand.Intn(l)
	return peers[i]
}
