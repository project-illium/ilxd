// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"context"
	"encoding/binary"
	ids "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/datastore"
	"github.com/project-illium/ilxd/types/blocks"
)

// IndexManager maintains the blockchain indexes and ensures they are current
// with the blockchain.
type IndexManager struct {
	indexers []Indexer
	ds       datastore.Datastore
}

// NewIndexManager returns a new IndexManager.
func NewIndexManager(ds datastore.Datastore, indexers []Indexer) *IndexManager {
	return &IndexManager{
		indexers: indexers,
		ds:       ds,
	}
}

// Init iterates over each indexer and checks to see if the indexer height is
// the same height as the tip of the chain. If not, it will roll the index
// forward until it is current.
func (im *IndexManager) Init(tipHeight uint32, getBlock func(height uint32) (*blocks.Block, error)) error {
	dbtx, err := im.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	for _, indexer := range im.indexers {
		height, err := dsFetchIndexHeight(dbtx, indexer)
		if err == ids.ErrNotFound { // New index
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
	}
	return nil
}

// Close shuts down all the indexers.
func (im *IndexManager) Close() error {
	for _, indexer := range im.indexers {
		if err := indexer.Close(im.ds); err != nil {
			return err
		}
	}
	return nil
}

func dsPutIndexerHeight(dbtx ids.Txn, indexer Indexer, height uint32) error {
	heightBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(heightBytes, height)
	return dbtx.Put(context.Background(), ids.NewKey(repo.IndexerHeightKeyPrefix+indexer.Key()), heightBytes)
}

func dsFetchIndexHeight(dbtx ids.Txn, indexer Indexer) (uint32, error) {
	heightBytes, err := dbtx.Get(context.Background(), ids.NewKey(repo.IndexerHeightKeyPrefix+indexer.Key()))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(heightBytes), nil
}

func dsPutIndexValue(dbtx ids.Txn, indexer Indexer, key string, value []byte) error {
	return dbtx.Put(context.Background(), ids.NewKey(repo.IndexKeyPrefix+indexer.Key()+"/"+key), value)
}

func dsFetchIndexValue(ds ids.Datastore, indexer Indexer, key string) ([]byte, error) {
	return ds.Get(context.Background(), ids.NewKey(repo.IndexKeyPrefix+indexer.Key()+"/"+key))
}

func dsFetchIndexValueWithTx(dbtx ids.Txn, indexer Indexer, key string) ([]byte, error) {
	return dbtx.Get(context.Background(), ids.NewKey(repo.IndexKeyPrefix+indexer.Key()+"/"+key))
}

func dsPrefixQueryIndexValue(dbtx ids.Txn, indexer Indexer, prefix string) (query.Results, error) {
	return dbtx.Query(context.Background(), query.Query{Prefix: repo.IndexKeyPrefix + indexer.Key() + "/" + prefix})
}

func dsDeleteIndexValue(dbtx ids.Txn, indexer Indexer, key string) error {
	return dbtx.Delete(context.Background(), ids.NewKey(repo.IndexKeyPrefix+indexer.Key()+"/"+key))
}

func dsDropIndex(ds datastore.Datastore, indexer Indexer) error {
	q := query.Query{
		Prefix: repo.IndexKeyPrefix + indexer.Key(),
	}
	dbtx, err := ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())
	results, err := dbtx.Query(context.Background(), q)
	if err != nil {
		return err
	}

	for result, ok := results.NextSync(); ok; result, ok = results.NextSync() {
		if err = dbtx.Delete(context.Background(), ids.NewKey(result.Key)); err != nil {
			return err
		}
	}
	return dbtx.Commit(context.Background())
}
