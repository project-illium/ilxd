// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"errors"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/project-illium/ilxd/net/pb"
	"github.com/project-illium/ilxd/repo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"time"
)

const (
	addrTTL       = time.Hour * 24 * 30
	gcInterval    = time.Hour * 2
	cacheInterval = time.Hour
)

type PeerstoreDS struct {
	ds     repo.Datastore
	pstore peerstore.Peerstore
	done   chan struct{}
}

func NewPeerstoreDS(ds repo.Datastore, pstore peerstore.Peerstore) *PeerstoreDS {
	pds := &PeerstoreDS{
		ds:     ds,
		pstore: pstore,
		done:   make(chan struct{}),
	}
	go pds.run()
	return pds
}

func (pds *PeerstoreDS) AddrInfos() ([]peer.AddrInfo, error) {
	var addrInfos []peer.AddrInfo
	query, err := pds.ds.Query(context.Background(), query.Query{
		Prefix: repo.CachedAddrInfoDatastoreKey,
	})
	if err != nil && errors.Is(err, datastore.ErrNotFound) {
		return addrInfos, nil
	}
	if err != nil {
		return nil, err
	}
	defer query.Close()

	for r := range query.Next() {
		var addrInfo pb.DBAddrInfo
		if err := proto.Unmarshal(r.Value, &addrInfo); err != nil {
			return nil, err
		}
		if time.Now().Before(addrInfo.LastSeen.AsTime().Add(addrTTL)) {
			s := strings.Split(r.Key, "/")
			ai := peer.AddrInfo{
				ID:    peer.ID(s[len(s)-1]),
				Addrs: make([]multiaddr.Multiaddr, 0, len(addrInfo.Addrs)),
			}
			for _, b := range addrInfo.Addrs {
				ma, err := multiaddr.NewMultiaddrBytes(b)
				if err != nil {
					return nil, err
				}
				ai.Addrs = append(ai.Addrs, ma)
			}
			addrInfos = append(addrInfos, ai)
		}
	}

	return addrInfos, nil
}

func (pds *PeerstoreDS) Close() {
	close(pds.done)
}

func (pds *PeerstoreDS) run() {
	cacheTicker := time.NewTicker(cacheInterval)
	gsTicker := time.NewTicker(gcInterval)
	time.AfterFunc(time.Minute, func() { pds.cachePeerAddrs() })
	for {
		select {
		case <-cacheTicker.C:
			if err := pds.cachePeerAddrs(); err != nil {
				log.Errorf("Error caching peerstore addrs: %s", err)
			}
		case <-gsTicker.C:
			if err := pds.garbageCollect(); err != nil {
				log.Errorf("Error garbage collecting peerstore addrs: %s", err)
			}
		case <-pds.done:
			if err := pds.garbageCollect(); err != nil {
				log.Errorf("Error garbage collecting peerstore addrs: %s", err)
			}
		}
	}
}

func (pds *PeerstoreDS) cachePeerAddrs() error {
	batch, err := pds.ds.Batch(context.Background())
	if err != nil {
		return err
	}
	for _, p := range pds.pstore.PeersWithAddrs() {
		addrs := pds.pstore.Addrs(p)
		a := &pb.DBAddrInfo{
			LastSeen: timestamppb.Now(),
			Addrs:    make([][]byte, 0, len(addrs)),
		}
		for _, addr := range addrs {
			b, err := addr.MarshalBinary()
			if err != nil {
				return err
			}
			a.Addrs = append(a.Addrs, b)
		}
		ser, err := proto.Marshal(a)
		if err != nil {
			return err
		}

		if err := batch.Put(context.Background(), datastore.NewKey(repo.CachedAddrInfoDatastoreKey+p.String()), ser); err != nil {
			return err
		}
	}
	return batch.Commit(context.Background())
}

func (pds *PeerstoreDS) garbageCollect() error {
	q := query.Query{
		Prefix: repo.CachedAddrInfoDatastoreKey,
	}
	query, err := pds.ds.Query(context.Background(), q)
	if err != nil {
		return err
	}

	var toDelete []string
	for r := range query.Next() {
		var addrInfo pb.DBAddrInfo
		if err := proto.Unmarshal(r.Value, &addrInfo); err != nil {
			return err
		}
		if time.Now().After(addrInfo.LastSeen.AsTime().Add(addrTTL)) {
			toDelete = append(toDelete, r.Key)
		}
	}

	query.Close()

	if len(toDelete) == 0 {
		return nil
	}

	batch, err := pds.ds.Batch(context.Background())
	if err != nil {
		return err
	}
	for _, key := range toDelete {
		if err := batch.Delete(context.Background(), datastore.NewKey(key)); err != nil {
			return err
		}
	}

	return batch.Commit(context.Background())
}
