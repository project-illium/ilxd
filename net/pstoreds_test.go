// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	pt "github.com/libp2p/go-libp2p/core/test"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func randomPeer(t *testing.T, addrCount int) peer.AddrInfo {
	var (
		pid   peer.ID
		err   error
		addrs = make([]ma.Multiaddr, addrCount)
		aFmt  = "/ip4/127.0.0.1/tcp/%d/ipfs/%s"
	)

	t.Helper()
	if pid, err = pt.RandPeerID(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < addrCount; i++ {
		if addrs[i], err = ma.NewMultiaddr(fmt.Sprintf(aFmt, i, pid)); err != nil {
			t.Fatal(err)
		}
	}
	return peer.AddrInfo{ID: pid, Addrs: addrs}
}

func TestDatastore(t *testing.T) {
	ds := mock.NewMapDatastore()
	pstore, err := pstoremem.NewPeerstore()
	assert.NoError(t, err)

	pstoreds := NewPeerstoreds(ds, pstore)

	peerMap := make(map[peer.ID]bool)
	for i := 0; i < 10; i++ {
		peer := randomPeer(t, 4)
		peerMap[peer.ID] = true
		pstore.AddAddrs(peer.ID, peer.Addrs, time.Hour)
	}

	addrInfos, err := pstoreds.AddrInfos()
	assert.NoError(t, err)
	assert.Len(t, addrInfos, 0)

	assert.NoError(t, pstoreds.cachePeerAddrs())

	addrInfos, err = pstoreds.AddrInfos()
	assert.NoError(t, err)
	assert.Len(t, addrInfos, 10)
	for _, a := range addrInfos {
		assert.True(t, peerMap[a.ID])
	}
}
