// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/walletlib"
	"strings"
	"sync"
	"time"
)

var _ Indexer = (*AddrIndex)(nil)

const (
	AddrIndexName = "address index"
	maxCacheSize  = 10000
)

type AddrIndex struct {
	outputIndex uint64
	nullifiers  map[types.Nullifier]walletlib.Address
	toDelete    map[types.Nullifier]bool
	height      uint32
	params      *params.NetworkParams
	ds          repo.Datastore
	stateMtx    sync.RWMutex
	quit        chan struct{}
}

var (
	publicAddrScriptHash       []byte
	publicAddrScriptCommitment []byte
)

// NewAddrIndex returns a new AddrIndex
func NewAddrIndex(ds repo.Datastore, networkParams *params.NetworkParams) (*AddrIndex, error) {
	outputIndex := uint64(0)
	val, err := dsFetchIndexValue(ds, &AddrIndex{}, repo.AddrIndexOutputIndexKey)
	if err != nil && !errors.Is(err, datastore.ErrNotFound) {
		return nil, err
	} else if err == nil {
		outputIndex = binary.BigEndian.Uint64(val)
	}

	publicAddrScriptHash = zk.PublicAddressScriptHash()
	publicAddrScriptCommitment = zk.PublicAddressScriptCommitment()
	a := &AddrIndex{
		nullifiers:  make(map[types.Nullifier]walletlib.Address),
		toDelete:    make(map[types.Nullifier]bool),
		params:      networkParams,
		stateMtx:    sync.RWMutex{},
		outputIndex: outputIndex,
		ds:          ds,
		quit:        make(chan struct{}),
	}
	go a.run()
	return a, nil
}

// Key returns the key of the index as a string.
func (idx *AddrIndex) Key() string {
	return repo.AddrIndexKey
}

// Name returns the human-readable name of the index.
func (idx *AddrIndex) Name() string {
	return AddrIndexName
}

// ConnectBlock is called when a block is connected to the chain.
// The indexer can use this opportunity to parse it and store it in
// the database. The database transaction must be respected.
func (idx *AddrIndex) ConnectBlock(dbtx datastore.Txn, blk *blocks.Block) error {
	idx.stateMtx.Lock()
	defer idx.stateMtx.Unlock()

	for _, tx := range blk.Transactions {
		for _, n := range tx.Nullifiers() {
			addr, err := idx.fetchUtxoAddress(dbtx, n)
			if err == nil {
				dsKey := repo.WalletServerTxKeyPrefix + addr.String() + "/" + tx.ID().String()
				if err := dsPutIndexValue(dbtx, idx, dsKey, nil); err != nil {
					return err
				}
				delete(idx.nullifiers, n)
				idx.toDelete[n] = true
			}
		}

		for _, out := range tx.Outputs() {
			idx.outputIndex++
			if len(out.Ciphertext) >= 160 && bytes.Equal(out.Ciphertext[0:32], publicAddrScriptHash) {
				note := &types.SpendNote{}
				if err := note.Deserialize(out.Ciphertext); err != nil {
					continue
				}
				if len(note.State) != 1 || len(note.State[0]) != 32 {
					continue
				}
				addr, err := walletlib.NewPublicAddressFromCommitment(note.State[0], idx.params)
				if err != nil {
					continue
				}

				nullifier, err := types.CalculateNullifier(idx.outputIndex-1, note.Salt, publicAddrScriptCommitment)
				if err != nil {
					continue
				}

				dsKey := repo.AddrIndexAddrKeyPrefix + addr.String() + "/" + tx.ID().String()
				if err := dsPutIndexValue(dbtx, idx, dsKey, nil); err != nil {
					return err
				}

				idx.nullifiers[nullifier] = addr
			}
		}
	}
	idx.height = blk.Header.Height
	go idx.maybeFlush()
	return nil
}

func (idx *AddrIndex) fetchUtxoAddress(dbtx datastore.Txn, n types.Nullifier) (walletlib.Address, error) {
	if addr, ok := idx.nullifiers[n]; ok {
		return addr, nil
	}
	if idx.toDelete[n] {
		return nil, errors.New("utxo not found")
	}

	val, err := dsFetchIndexValueWithTx(dbtx, idx, repo.AddrIndexNulliferKeyPrefix+n.String())
	if err != nil {
		return nil, err
	}
	return walletlib.DecodeAddress(string(val), idx.params)
}

// GetTransactionsIDs returns the transaction IDs stored for the given address
func (idx *AddrIndex) GetTransactionsIDs(ds repo.Datastore, addr walletlib.Address) ([]types.ID, error) {
	dbtx, err := ds.NewTransaction(context.Background(), false)
	if err != nil {
		return nil, err
	}
	defer dbtx.Discard(context.Background())

	query, err := dsPrefixQueryIndexValue(dbtx, &AddrIndex{}, repo.AddrIndexAddrKeyPrefix+addr.String())
	if err != nil {
		return nil, err
	}

	var txids []types.ID
	for r := range query.Next() {
		v := strings.Split(r.Key, "/")
		txidStr := v[len(v)-1]

		txid, err := types.NewIDFromString(txidStr)
		if err != nil {
			return nil, err
		}
		txids = append(txids, txid)
	}

	return txids, dbtx.Commit(context.Background())
}

func (idx *AddrIndex) maybeFlush() {
	idx.stateMtx.Lock()
	defer idx.stateMtx.Unlock()

	if len(idx.nullifiers)+len(idx.toDelete) > maxCacheSize {
		if err := idx.flush(); err != nil {
			log.WithCaller(true).Error("Address index error flushing to disk", log.Args("error", err))
		}
	}
}

func (idx *AddrIndex) flush() error {
	dbtx, err := idx.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}

	for n, addr := range idx.nullifiers {
		dsKey := repo.AddrIndexNulliferKeyPrefix + n.String()
		if err := dsPutIndexValue(dbtx, idx, dsKey, []byte(addr.String())); err != nil {
			return err
		}
	}
	for n := range idx.toDelete {
		dsKey := repo.AddrIndexNulliferKeyPrefix + n.String()
		if err := dsDeleteIndexValue(dbtx, idx, dsKey); err != nil {
			return err
		}
	}
	outputsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(outputsBytes, idx.outputIndex)

	if err := dsPutIndexValue(dbtx, idx, repo.AddrIndexOutputIndexKey, outputsBytes); err != nil {
		return err
	}

	if err := dsPutIndexerHeight(dbtx, idx, idx.height); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	idx.toDelete = make(map[types.Nullifier]bool)

	// Delete half of the cache
	count := len(idx.nullifiers) / 2
	for n := range idx.nullifiers {
		delete(idx.nullifiers, n)
		count--
		if count <= 0 {
			break
		}
	}
	return nil
}

func (idx *AddrIndex) run() {
	flushTicker := time.NewTicker(flushTickerInterval)
	for {
		select {
		case <-idx.quit:
			return
		case <-flushTicker.C:
			idx.stateMtx.Lock()
			if err := idx.flush(); err != nil {
				log.WithCaller(true).Error("Address index error flushing to disk", log.Args("error", err))
			}
			idx.stateMtx.Unlock()
		}
	}
}

// Close is called when the index manager shuts down and gives the indexer
// an opportunity to do some cleanup.
func (idx *AddrIndex) Close(ds repo.Datastore) error {
	close(idx.quit)

	return idx.flush()
}

func DropAddrIndex(ds repo.Datastore) error {
	return dsDropIndex(ds, &AddrIndex{})
}
