// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mock

import (
	"context"
	"errors"
	ids "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/blockstore"
	"github.com/project-illium/ilxd/repo/datastore"
)

type mockDatastore struct {
	*mapDatastore
	*blockstore.FlatFilestore
}

func NewMockDatastore() datastore.Datastore {
	ds := NewMapDatastore()
	bs, _ := blockstore.NewMockFlatFilestore(&params.RegestParams)
	return &mockDatastore{
		mapDatastore:  ds,
		FlatFilestore: bs,
	}
}

func (ds *mockDatastore) NewTransaction(ctx context.Context, readOnly bool) (datastore.Txn, error) {
	dbtx, err := ds.mapDatastore.NewTransaction(ctx, readOnly)
	if err != nil {
		return nil, err
	}

	btx, err := ds.FlatFilestore.NewTransaction(ctx, readOnly)
	if err != nil {
		return nil, err
	}
	return &txn{
		Txn:    dbtx,
		BlkTxn: btx,
	}, nil
}

type txn struct {
	ids.Txn
	*blockstore.BlkTxn
}

func (dbtx *txn) Commit(ctx context.Context) error {
	if err := dbtx.Txn.Commit(ctx); err != nil {
		return err
	}

	return dbtx.BlkTxn.Commit(ctx)
}

func (dbtx *txn) Discard(ctx context.Context) {
	dbtx.Txn.Discard(ctx)
	dbtx.BlkTxn.Discard(context.Background())
}

type mapDatastore struct {
	ids.MapDatastore
}

func NewMapDatastore() *mapDatastore {
	ds := ids.NewMapDatastore()
	return &mapDatastore{MapDatastore: *ds}
}

func (ds *mapDatastore) DiskUsage(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (ds *mapDatastore) NewTransaction(ctx context.Context, readOnly bool) (ids.Txn, error) {
	return &mapTxn{
		readOnly: readOnly,
		ds:       ds,
		puts:     make(map[ids.Key][]byte),
		deletes:  make(map[ids.Key]struct{}),
	}, nil
}

type mapTxn struct {
	readOnly bool
	ds       *mapDatastore
	puts     map[ids.Key][]byte
	deletes  map[ids.Key]struct{}
}

func (t *mapTxn) Get(ctx context.Context, key ids.Key) (value []byte, err error) {
	return t.ds.Get(ctx, key)
}

func (t *mapTxn) Has(ctx context.Context, key ids.Key) (exists bool, err error) {
	return t.ds.Has(ctx, key)
}

func (t *mapTxn) GetSize(ctx context.Context, key ids.Key) (size int, err error) {
	return t.ds.GetSize(ctx, key)
}

func (t *mapTxn) Query(ctx context.Context, q query.Query) (query.Results, error) {
	return t.ds.Query(ctx, q)
}

func (t *mapTxn) Put(ctx context.Context, key ids.Key, value []byte) error {
	if t.readOnly {
		return errors.New("transaction is read only")
	}
	t.puts[key] = value
	return nil
}

func (t *mapTxn) Delete(ctx context.Context, key ids.Key) error {
	if t.readOnly {
		return errors.New("transaction is read only")
	}
	t.deletes[key] = struct{}{}
	return nil
}

func (t *mapTxn) Commit(ctx context.Context) error {
	for k, v := range t.puts {
		t.ds.Put(ctx, k, v)
	}
	for k := range t.deletes {
		t.ds.Delete(ctx, k)
	}
	return nil
}

func (t *mapTxn) Discard(ctx context.Context) {
	t.puts = make(map[ids.Key][]byte)
	t.deletes = make(map[ids.Key]struct{})
}
