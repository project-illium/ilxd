// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	mrand "math/rand"
	"testing"
)

type mockChooser2 struct {
	peers []peer.ID
}

func (m *mockChooser2) WeightedRandomValidator() peer.ID {
	i := mrand.Intn(len(m.peers))
	return m.peers[i]
}

func TestBackoffChooser(t *testing.T) {
	chooser := &mockChooser2{
		peers: make([]peer.ID, 10),
	}
	for i := 0; i < 10; i++ {
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		assert.NoError(t, err)
		pid, err := peer.IDFromPrivateKey(priv)
		assert.NoError(t, err)
		chooser.peers[i] = pid
	}

	bochooser := NewBackoffChooser(chooser, &MockValConn{})

	for i := 0; i < 100; i++ {
		assert.NotEqual(t, "", bochooser.WeightedRandomValidator())
	}

	bochooser.RegisterDialFailure(chooser.peers[0])
	for i := 0; i < 100; i++ {
		assert.NotEqual(t, chooser.peers[0], bochooser.WeightedRandomValidator())
	}

	bochooser.RegisterDialSuccess(chooser.peers[0])
	chosen := false
	for i := 0; i < 100; i++ {
		p := bochooser.WeightedRandomValidator()
		if p == chooser.peers[0] {
			chosen = true
		}
	}
	assert.True(t, chosen)
}
