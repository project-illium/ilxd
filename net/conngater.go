// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/project-illium/ilxd/repo"
	"net"
	"path"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

// ConnectionGater implements a connection gater that allows the application to perform
// access control on incoming and outgoing connections.
type ConnectionGater struct {
	sync.RWMutex

	scores map[peer.ID]*DynamicBanScore

	blockedPeers map[peer.ID]time.Time
	blockedAddrs map[string]time.Time

	banDuration time.Duration
	maxBanscore uint32

	ds       repo.Datastore
	addrBook peerstore.AddrBook
}

const (
	keyPeer = "peer/"
	keyAddr = "addr/"
)

// NewConnectionGater creates a new connection gater.
// The ds argument is an (optional, can be nil) datastore to persist the connection gater
// filters.
func NewConnectionGater(ds datastore.Datastore, addrBook peerstore.AddrBook, banDuration time.Duration, maxBanscore uint32) (*ConnectionGater, error) {
	cg := &ConnectionGater{
		blockedPeers: make(map[peer.ID]time.Time),
		blockedAddrs: make(map[string]time.Time),
		scores:       make(map[peer.ID]*DynamicBanScore),
		banDuration:  banDuration,
		maxBanscore:  maxBanscore,
		addrBook:     addrBook,
	}

	if ds != nil {
		err := cg.loadRules(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return cg, nil
}

func (cg *ConnectionGater) loadRules(ctx context.Context) error {
	// load blocked peers
	res, err := cg.ds.Query(ctx, query.Query{Prefix: keyPeer})
	if err != nil {
		log.Errorf("error querying datastore for blocked peers: %s", err)
		return err
	}

	for r := range res.Next() {
		if r.Error != nil {
			log.Errorf("query result error: %s", r.Error)
			return err
		}

		p := peer.ID(path.Base(r.Key))
		t := time.Time{}
		if err = t.GobDecode(r.Entry.Value); err != nil {
			log.Errorf("time deserialization error: %s", r.Error)
			return err
		}
		cg.blockedPeers[p] = t
		time.AfterFunc(t.Sub(time.Now()), func() {
			if err := cg.UnblockPeer(p); err != nil {
				log.Errorf("error unblocking peer after expiration: %s", err)
			}
		})
	}

	// load blocked addrs
	res, err = cg.ds.Query(ctx, query.Query{Prefix: keyAddr})
	if err != nil {
		log.Errorf("error querying datastore for blocked addrs: %s", err)
		return err
	}

	for r := range res.Next() {
		if r.Error != nil {
			log.Errorf("query result error: %s", r.Error)
			return err
		}

		ip := net.IP(path.Base(r.Key))
		t := time.Time{}
		if err = t.GobDecode(r.Entry.Value); err != nil {
			log.Errorf("time deserialization error: %s", r.Error)
			return err
		}
		cg.blockedAddrs[ip.String()] = t
		time.AfterFunc(t.Sub(time.Now()), func() {
			if err := cg.UnblockAddr(ip); err != nil {
				log.Errorf("error unblocking addr after expiration: %s", err)
			}
		})
	}

	return nil
}

func (cg *ConnectionGater) IncreaseBanscore(p peer.ID, persistent, transient uint32) error {
	if cg.banDuration == 0 {
		return nil
	}
	cg.Lock()
	banscore, ok := cg.scores[p]
	if !ok {
		cg.scores[p] = &DynamicBanScore{}
	}
	score := banscore.Increase(persistent, transient)
	cg.Unlock()
	if score > cg.maxBanscore {
		if err := cg.BlockPeer(p); err != nil {
			return err
		}

		addrs := cg.addrBook.Addrs(p)
		for _, addr := range addrs {
			ip, err := manet.ToIP(addr)
			if err == nil {
				if err := cg.BlockAddr(ip); err != nil {
					return err
				}
			}
		}
	}
	cg.Lock()
	defer cg.Unlock()
	for p, s := range cg.scores {
		if s.Int() == 0 {
			delete(cg.scores, p)
		}
	}
	return nil
}

// BlockPeer adds a peer to the set of blocked peers.
// Note: active connections to the peer are not automatically closed.
func (cg *ConnectionGater) BlockPeer(p peer.ID) error {
	banExpiration := time.Now().Add(cg.banDuration)
	b, err := banExpiration.GobEncode()
	if err != nil {
		return err
	}
	if cg.ds != nil {
		err := cg.ds.Put(context.Background(), datastore.NewKey(repo.ConnGaterKeyPrefix+keyPeer+p.String()), b)
		if err != nil {
			log.Errorf("error writing blocked peer to datastore: %s", err)
			return err
		}
	}

	cg.Lock()
	defer cg.Unlock()
	cg.blockedPeers[p] = banExpiration

	time.AfterFunc(cg.banDuration, func() {
		if err := cg.UnblockPeer(p); err != nil {
			log.Errorf("error unblocking peer after expiration: %s", err)
		}
	})

	return nil
}

// UnblockPeer removes a peer from the set of blocked peers
func (cg *ConnectionGater) UnblockPeer(p peer.ID) error {
	if cg.ds != nil {
		err := cg.ds.Delete(context.Background(), datastore.NewKey(repo.ConnGaterKeyPrefix+keyPeer+p.String()))
		if err != nil {
			log.Errorf("error deleting blocked peer from datastore: %s", err)
			return err
		}
	}

	cg.Lock()
	defer cg.Unlock()

	delete(cg.blockedPeers, p)

	return nil
}

// ListBlockedPeers return a list of blocked peers
func (cg *ConnectionGater) ListBlockedPeers() []peer.ID {
	cg.RLock()
	defer cg.RUnlock()

	result := make([]peer.ID, 0, len(cg.blockedPeers))
	for p := range cg.blockedPeers {
		result = append(result, p)
	}

	return result
}

// BlockAddr adds an IP address to the set of blocked addresses.
// Note: active connections to the IP address are not automatically closed.
func (cg *ConnectionGater) BlockAddr(ip net.IP) error {
	banExpiration := time.Now().Add(cg.banDuration)
	b, err := banExpiration.GobEncode()
	if err != nil {
		return err
	}
	if cg.ds != nil {
		err := cg.ds.Put(context.Background(), datastore.NewKey(repo.ConnGaterKeyPrefix+keyAddr+ip.String()), b)
		if err != nil {
			log.Errorf("error writing blocked addr to datastore: %s", err)
			return err
		}
	}

	cg.Lock()
	defer cg.Unlock()

	cg.blockedAddrs[ip.String()] = banExpiration
	time.AfterFunc(cg.banDuration, func() {
		if err := cg.UnblockAddr(ip); err != nil {
			log.Errorf("error unblocking addr after expiration: %s", err)
		}
	})

	return nil
}

// UnblockAddr removes an IP address from the set of blocked addresses
func (cg *ConnectionGater) UnblockAddr(ip net.IP) error {
	if cg.ds != nil {
		err := cg.ds.Delete(context.Background(), datastore.NewKey(repo.ConnGaterKeyPrefix+keyAddr+ip.String()))
		if err != nil {
			log.Errorf("error deleting blocked addr from datastore: %s", err)
			return err
		}
	}

	cg.Lock()
	defer cg.Unlock()

	delete(cg.blockedAddrs, ip.String())

	return nil
}

// ListBlockedAddrs return a list of blocked IP addresses
func (cg *ConnectionGater) ListBlockedAddrs() []net.IP {
	cg.RLock()
	defer cg.RUnlock()

	result := make([]net.IP, 0, len(cg.blockedAddrs))
	for ipStr := range cg.blockedAddrs {
		ip := net.ParseIP(ipStr)
		result = append(result, ip)
	}

	return result
}

// ConnectionGater interface
var _ connmgr.ConnectionGater = (*ConnectionGater)(nil)

func (cg *ConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	cg.RLock()
	defer cg.RUnlock()

	_, block := cg.blockedPeers[p]
	return !block
}

func (cg *ConnectionGater) InterceptAddrDial(p peer.ID, a ma.Multiaddr) (allow bool) {
	// we have already filtered blocked peers in InterceptPeerDial, so we just check the IP
	cg.RLock()
	defer cg.RUnlock()

	ip, err := manet.ToIP(a)
	if err != nil {
		log.Warnf("error converting multiaddr to IP addr: %s", err)
		return true
	}

	_, block := cg.blockedAddrs[ip.String()]
	if block {
		return false
	}

	return true
}

func (cg *ConnectionGater) InterceptAccept(cma network.ConnMultiaddrs) (allow bool) {
	cg.RLock()
	defer cg.RUnlock()

	a := cma.RemoteMultiaddr()

	ip, err := manet.ToIP(a)
	if err != nil {
		log.Warnf("error converting multiaddr to IP addr: %s", err)
		return true
	}

	_, block := cg.blockedAddrs[ip.String()]
	if block {
		return false
	}

	return true
}

func (cg *ConnectionGater) InterceptSecured(dir network.Direction, p peer.ID, cma network.ConnMultiaddrs) (allow bool) {
	if dir == network.DirOutbound {
		// we have already filtered those in InterceptPeerDial/InterceptAddrDial
		return true
	}

	// we have already filtered addrs in InterceptAccept, so we just check the peer ID
	cg.RLock()
	defer cg.RUnlock()

	_, block := cg.blockedPeers[p]
	return !block
}

func (cg *ConnectionGater) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
