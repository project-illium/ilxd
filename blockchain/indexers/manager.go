// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"context"
	"encoding/binary"
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types/blocks"
)

// IndexManager maintains the blockchain indexes and ensures they are current
// with the blockchain.
type IndexManager struct {
	indexers []Indexer
	ds       repo.Datastore
}

// NewIndexManager returns a new IndexManager.
func NewIndexManager(ds repo.Datastore, indexers []Indexer) *IndexManager {
	return &IndexManager{
		indexers: indexers,
		ds:       ds,
	}
}

// Init iterates over each indexer and checks to see if the indexer height is
// the same height as the tip of the chain. If not, it will roll the index
// forward until it is current.
func (im *IndexManager) Init(tipHeight uint32, getBlock GetBlockFunc) error {
	dbtx, err := im.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	for _, indexer := range im.indexers {
		height, err := dsFetchIndexHeight(dbtx, indexer)
		if err == datastore.ErrNotFound { // New index
			genesis, err := getBlock(0)
			if err != nil {
				return err
			}
			if err := indexer.ConnectBlock(dbtx, genesis); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		if height < tipHeight {
			for n := height + 1; n <= tipHeight; n++ {
				blk, err := getBlock(n)
				if err != nil {
					return err
				}
				if err := indexer.ConnectBlock(dbtx, blk); err != nil {
					return err
				}
			}

			if err := dsPutIndexerHeight(dbtx, indexer, tipHeight); err != nil {
				return err
			}
		}
	}
	return dbtx.Commit(context.Background())
}

// ConnectBlock connects the block to each indexer.
func (im *IndexManager) ConnectBlock(dbtx datastore.Txn, blk *blocks.Block) error {
	for _, indexer := range im.indexers {
		if err := indexer.ConnectBlock(dbtx, blk); err != nil {
			return err
		}
		if err := dsPutIndexerHeight(dbtx, indexer, blk.Header.Height); err != nil {
			return err
		}
	}
	return nil
}

func dsPutIndexerHeight(dbtx datastore.Txn, indexer Indexer, height uint32) error {
	heightBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(heightBytes, height)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.IndexerHeightKeyPrefix+indexer.Key()), heightBytes)
}

func dsFetchIndexHeight(dbtx datastore.Txn, indexer Indexer) (uint32, error) {
	heightBytes, err := dbtx.Get(context.Background(), datastore.NewKey(repo.IndexerHeightKeyPrefix+indexer.Key()))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(heightBytes), nil
}

func dsPutIndexValue(dbtx datastore.Txn, indexer Indexer, key string, value []byte) error {
	return dbtx.Put(context.Background(), datastore.NewKey(repo.IndexKeyPrefix+indexer.Key()+"/"+key), value)
}

func dsFetchIndexValue(ds repo.Datastore, indexer Indexer, key string) ([]byte, error) {
	return ds.Get(context.Background(), datastore.NewKey(repo.IndexKeyPrefix+indexer.Key()+"/"+key))
}
