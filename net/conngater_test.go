// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"

	ma "github.com/multiformats/go-multiaddr"
)

func TestConnectionGater(t *testing.T) {
	ds := mock.NewMapDatastore()
	pstore, err := pstoreds.NewPeerstore(context.Background(), ds, pstoreds.DefaultOpts())
	assert.NoError(t, err)

	peerA, _ := peer.Decode("12D3KooWSE3nPEMZEXGpDRjZesMEVquvs3YjYPJdiC4ve66rVuu5")
	peerB, _ := peer.Decode("12D3KooWARnj9CFGko6iX3PV8sYfMG94SSbMtTm7XPjtiiKjV7Fs")

	ip1 := net.ParseIP("1.2.3.4")

	cg, err := NewConnectionGater(ds, pstore, time.Minute, 100)
	assert.NoError(t, err)

	// test peer blocking
	allow := cg.InterceptPeerDial(peerA)
	assert.True(t, allow, "expected gater to allow peerA")

	allow = cg.InterceptPeerDial(peerB)
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerB")

	err = cg.BlockPeer(peerA)
	assert.NoError(t, err)

	allow = cg.InterceptPeerDial(peerA)
	assert.False(t, allow, "expected gater to deny peerA")

	allow = cg.InterceptPeerDial(peerB)
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.False(t, allow, "expected gater to deny peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerB")

	// test addr and subnet blocking
	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	err = cg.BlockAddr(ip1)
	assert.NoError(t, err)

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.False(t, allow, "expected gater to deny peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.False(t, allow, "expected gater to deny peerB in 1.2.3.4")

	// make a new gater reusing the datastore to test persistence
	cg, err = NewConnectionGater(ds, pstore, time.Minute, 100)
	assert.NoError(t, err)

	// test the list methods while at it
	blockedPeers := cg.ListBlockedPeers()
	assert.Equalf(t, 1, len(blockedPeers), "expected 1 blocked peer, but got %d", len(blockedPeers))

	blockedAddrs := cg.ListBlockedAddrs()
	assert.Equalf(t, 1, len(blockedAddrs), "expected 1 blocked addr, but got %d", len(blockedAddrs))

	allow = cg.InterceptPeerDial(peerA)
	assert.False(t, allow, "expected gater to deny peerA")

	allow = cg.InterceptPeerDial(peerB)
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.False(t, allow, "expected gater to deny peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.False(t, allow, "expected gater to block peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.False(t, allow, "expected gater to deny peerB in 1.2.3.4")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	// undo the blocks to ensure that we can unblock stuff
	err = cg.UnblockPeer(peerA)
	assert.NoError(t, err)

	err = cg.UnblockAddr(ip1)
	assert.NoError(t, err)

	allow = cg.InterceptPeerDial(peerA)
	assert.True(t, allow, "expected gater to allow peerA")

	allow = cg.InterceptPeerDial(peerB)
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	// make a new gater reusing the datastore to test persistence of unblocks
	cg, err = NewConnectionGater(ds, pstore, time.Minute, 100)
	assert.NoError(t, err)

	allow = cg.InterceptPeerDial(peerA)
	assert.True(t, allow, "expected gater to allow peerA")

	allow = cg.InterceptPeerDial(peerB)
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.True(t, allow, "expected gater to allow peerB")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.True(t, allow, "expected gater to allow peerB in 2.3.4.5")
}

func TestBanscore(t *testing.T) {
	ds := mock.NewMapDatastore()
	pstore, err := pstoreds.NewPeerstore(context.Background(), ds, pstoreds.DefaultOpts())
	assert.NoError(t, err)

	peerA, _ := peer.Decode("12D3KooWSE3nPEMZEXGpDRjZesMEVquvs3YjYPJdiC4ve66rVuu5")

	addr := ma.StringCast("/ip4/1.2.3.4/tcp/1234")

	pstore.AddAddr(peerA, addr, time.Hour)

	cg, err := NewConnectionGater(ds, pstore, time.Minute, 100)
	assert.NoError(t, err)

	banned, err := cg.IncreaseBanscore(peerA, 99, 0)
	assert.NoError(t, err)
	assert.False(t, banned)

	banned, err = cg.IncreaseBanscore(peerA, 2, 0)
	assert.NoError(t, err)
	assert.True(t, banned)

	blockedPeers := cg.ListBlockedPeers()
	assert.Equalf(t, 1, len(blockedPeers), "expected 1 blocked peer, but got %d", len(blockedPeers))

	blockedAddrs := cg.ListBlockedAddrs()
	assert.Equalf(t, 1, len(blockedAddrs), "expected 1 blocked addr, but got %d", len(blockedAddrs))

	assert.Equal(t, peerA.String(), blockedPeers[0].String())
	assert.Equal(t, "1.2.3.4", blockedAddrs[0].String())
}

type mockConnMultiaddrs struct {
	local, remote ma.Multiaddr
}

func (cma *mockConnMultiaddrs) LocalMultiaddr() ma.Multiaddr {
	return cma.local
}

func (cma *mockConnMultiaddrs) RemoteMultiaddr() ma.Multiaddr {
	return cma.remote
}
