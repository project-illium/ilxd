// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package datastore

import (
	"context"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/blockstore"
	"os"
	"path"
)

type TxnDatastore interface {
	NewTransaction(ctx context.Context, readOnly bool) (Txn, error)
}

type Datastore interface {
	datastore.Datastore
	datastore.Batching
	datastore.PersistentDatastore
	blockstore.Blockstore
	TxnDatastore
}

type Txn interface {
	datastore.Txn

	// PutBlockData puts the block data, usually serialized transactions,
	// to the datastore.
	PutBlockData(height uint32, blockData []byte) (blockstore.BlockLocation, error)

	// FetchBlockData returns the block data from the datastore
	FetchBlockData(location blockstore.BlockLocation) ([]byte, error)

	// DeleteBefore deletes all block data before the provided height
	DeleteBefore(height uint32) error
}

type IlxdDatastore struct {
	*badger.Datastore
	*blockstore.FlatFilestore
}

func (ds *IlxdDatastore) Close() error {
	if err := ds.Datastore.Close(); err != nil {
		return err
	}
	return ds.FlatFilestore.Close()
}

func (ds *IlxdDatastore) NewTransaction(ctx context.Context, readOnly bool) (Txn, error) {
	dbtx, err := ds.Datastore.NewTransaction(ctx, readOnly)
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
	datastore.Txn
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

func NewIlxdDatastore(dataDir string, opts ...Option) (Datastore, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	if cfg.params == nil {
		cfg.params = &params.RegestParams
	}

	badgerOpts := &badger.DefaultOptions
	badgerOpts.MaxTableSize = 256 << 20
	ds, err := badger.NewDatastore(dataDir, badgerOpts)
	if err != nil {
		return nil, err
	}

	blocksDir := path.Join(dataDir, "blocks")
	if cfg.noBlockstore {
		// Empty blocks directory stops it from persisting
		// data to disk.
		blocksDir = ""
	} else {
		if _, err := os.Stat(blocksDir); os.IsNotExist(err) {
			err := os.MkdirAll(blocksDir, 0700)
			if err != nil {
				return nil, err
			}
		}
	}

	bs, err := blockstore.NewFlatFilestore(blocksDir, cfg.params)
	if err != nil {
		return nil, err
	}

	return &IlxdDatastore{
		Datastore:     ds,
		FlatFilestore: bs,
	}, nil
}
