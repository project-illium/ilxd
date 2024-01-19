// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mock

import (
	"context"
	"errors"
	datastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/project-illium/ilxd/repo"
)

var _ repo.Datastore = (*MapDatastore)(nil)

type MapDatastore struct {
	datastore.MapDatastore
}

func NewMapDatastore() *MapDatastore {
	ds := datastore.NewMapDatastore()
	return &MapDatastore{MapDatastore: *ds}
}

func (ds *MapDatastore) DiskUsage(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (ds *MapDatastore) NewTransaction(ctx context.Context, readOnly bool) (datastore.Txn, error) {
	return &txn{
		readOnly: readOnly,
		ds:       ds,
		puts:     make(map[datastore.Key][]byte),
		deletes:  make(map[datastore.Key]struct{}),
	}, nil
}

type txn struct {
	readOnly bool
	ds       *MapDatastore
	puts     map[datastore.Key][]byte
	deletes  map[datastore.Key]struct{}
}

func (t *txn) Get(ctx context.Context, key datastore.Key) (value []byte, err error) {
	return t.ds.Get(ctx, key)
}

func (t *txn) Has(ctx context.Context, key datastore.Key) (exists bool, err error) {
	return t.ds.Has(ctx, key)
}

func (t *txn) GetSize(ctx context.Context, key datastore.Key) (size int, err error) {
	return t.ds.GetSize(ctx, key)
}

func (t *txn) Query(ctx context.Context, q query.Query) (query.Results, error) {
	return t.ds.Query(ctx, q)
}

func (t *txn) Put(ctx context.Context, key datastore.Key, value []byte) error {
	if t.readOnly {
		return errors.New("transaction is read only")
	}
	t.puts[key] = value
	return nil
}

func (t *txn) Delete(ctx context.Context, key datastore.Key) error {
	if t.readOnly {
		return errors.New("transaction is read only")
	}
	t.deletes[key] = struct{}{}
	return nil
}

func (t *txn) Commit(ctx context.Context) error {
	for k, v := range t.puts {
		t.ds.Put(ctx, k, v)
	}
	for k := range t.deletes {
		t.ds.Delete(ctx, k)
	}
	return nil
}

func (t *txn) Discard(ctx context.Context) {
	t.puts = make(map[datastore.Key][]byte)
	t.deletes = make(map[datastore.Key]struct{})
}
