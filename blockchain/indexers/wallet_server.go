// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package indexers

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/walletlib"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var _ Indexer = (*WalletServerIndex)(nil)

const (
	staleUserThreshold      = time.Hour * 24 * 90
	staleUserTickerInterval = time.Hour * 24
	flushTickerInterval     = time.Hour * 10
)

type UserTransaction struct {
	Tx      *transactions.Transaction
	ViewKey crypto.PrivKey
}

type Subscription struct {
	C     chan *UserTransaction
	id    uint64
	Close func()
}

const (
	walletServerAccumulatorKey      = "accumulator"
	walletServerBestBlockKey        = "bestblockid"
	walletServerViewKeyPrefix       = "viewkey/"
	walletServerLockingScriptPrefix = "lockingscript/"
	walletServerNullifierKeyPrefix  = "nullifier/"
	walletServerTxKeyPrefix         = "tx/"
	walletServerIndexKey            = "walletserverindex"
	WalletServerIndexName           = "wallet server index"
)

type commitmentWithKey struct {
	commitment types.ID
	viewKey    crypto.PrivKey
}

// WalletServerIndex is and implementation of the Indexer which indexes
// transactions on behalf of wallets. It allows for building lite wallets
// that tradeoff privacy vis-à-vis the server for fast syncing and instant
// access to coins.
type WalletServerIndex struct {
	acc             *blockchain.Accumulator
	scanner         *walletlib.TransactionScanner
	nullifiers      map[types.Nullifier]commitmentWithKey
	stateMtx        sync.RWMutex
	bestBlockID     types.ID
	bestBlockHeight uint32
	subs            map[uint64]*Subscription
	subMtx          sync.RWMutex
	quit            chan struct{}
}

// NewWalletServerIndex returns a new WalletServerIndex.
func NewWalletServerIndex(ds repo.Datastore) (*WalletServerIndex, error) {
	dbtx, err := ds.NewTransaction(context.Background(), true)
	if err != nil {
		return nil, err
	}
	query, err := dsPrefixQueryIndexValue(dbtx, &WalletServerIndex{}, walletServerViewKeyPrefix)
	if err != nil {
		return nil, err
	}

	scanner := walletlib.NewTransactionScanner()

	for r := range query.Next() {
		v := strings.Split(r.Key, "/")
		keyBytes, err := hex.DecodeString(v[len(v)-1])
		if err != nil {
			return nil, err
		}

		key, err := crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}

		if _, ok := key.(*icrypto.Curve25519PrivateKey); !ok {
			return nil, errors.New("viewkey is not curve25519 private key")
		}
		scanner.AddKeys(key.(*icrypto.Curve25519PrivateKey))
	}
	query.Close()

	query, err = dsPrefixQueryIndexValue(dbtx, &WalletServerIndex{}, walletServerNullifierKeyPrefix)
	if err != nil {
		return nil, err
	}

	nullifiers := make(map[types.Nullifier]commitmentWithKey)
	for r := range query.Next() {
		v := strings.Split(r.Key, "/")
		nullifierStr := v[len(v)-1]
		keyBytes, err := hex.DecodeString(v[len(v)-2])
		if err != nil {
			return nil, err
		}

		nullifier, err := types.NewNullifierFromString(nullifierStr)
		if err != nil {
			return nil, err
		}

		key, err := crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}
		nullifiers[nullifier] = commitmentWithKey{
			commitment: types.NewID(r.Value),
			viewKey:    key,
		}
	}
	query.Close()

	var acc *blockchain.Accumulator
	height, err := dsFetchIndexHeight(dbtx, &WalletServerIndex{})
	if err := dbtx.Commit(context.Background()); err != nil {
		return nil, err
	}

	if err == datastore.ErrNotFound {
		acc = blockchain.NewAccumulator()
	} else if height > 0 {
		accBytes, err := dsFetchIndexValue(ds, &WalletServerIndex{}, walletServerAccumulatorKey)
		if err != nil {
			return nil, err
		}
		acc, err = blockchain.DeserializeAccumulator(accBytes)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	bestBlock, err := dsFetchIndexValue(ds, &WalletServerIndex{}, walletServerBestBlockKey)
	if err != nil && err != datastore.ErrNotFound {
		return nil, err
	} else if err == datastore.ErrNotFound {
		bestBlock = make([]byte, 36)
	}

	idx := &WalletServerIndex{
		acc:             acc,
		scanner:         scanner,
		bestBlockID:     types.NewID(bestBlock[4:]),
		bestBlockHeight: binary.BigEndian.Uint32(bestBlock[:4]),
		nullifiers:      nullifiers,
		subs:            make(map[uint64]*Subscription),
		quit:            make(chan struct{}),
		stateMtx:        sync.RWMutex{},
		subMtx:          sync.RWMutex{},
	}
	go idx.run(ds)
	return idx, nil
}

// Key returns the key of the index as a string.
func (idx *WalletServerIndex) Key() string {
	return walletServerIndexKey
}

// Name returns the human-readable name of the index.
func (idx *WalletServerIndex) Name() string {
	return WalletServerIndexName
}

// ConnectBlock is called when a block is connected to the chain.
// The indexer can use this opportunity to parse it and store it in
// the database. The database transaction must be respected.
func (idx *WalletServerIndex) ConnectBlock(dbtx datastore.Txn, blk *blocks.Block) error {
	idx.stateMtx.Lock()
	defer idx.stateMtx.Unlock()

	matches := idx.scanner.ScanOutputs(blk)

	for _, tx := range blk.Transactions {
		notifiedKeys := make(map[crypto.PrivKey]bool)
		for _, out := range tx.Outputs() {
			match, ok := matches[types.NewID(out.Commitment)]
			if ok {
				commitmentIndex := idx.acc.NumElements()
				idx.acc.Insert(out.Commitment, true)
				viewKey, err := crypto.MarshalPrivateKey(match.Key)
				if err != nil {
					return err
				}
				serializedViewKey := hex.EncodeToString(viewKey)
				dsKey := walletServerTxKeyPrefix + serializedViewKey + "/" + tx.ID().String()
				if err := dsPutIndexValue(dbtx, idx, dsKey, nil); err != nil {
					return err
				}

				var note types.SpendNote
				if err := note.Deserialize(match.DecryptedNote); err != nil {
					continue
				}

				serializedLockingScript, err := dsFetchIndexValueWithTx(dbtx, idx, walletServerLockingScriptPrefix+hex.EncodeToString(viewKey))
				if err != nil {
					continue
				}
				ul := new(types.LockingScript)
				if err := ul.Deserialize(serializedLockingScript); err != nil {
					return err
				}
				scriptHash, err := ul.Hash()
				if err != nil {
					return err
				}
				if scriptHash.Compare(note.ScriptHash) != 0 {
					continue
				}

				nullifier, err := types.CalculateNullifier(commitmentIndex, note.Salt, ul.ScriptCommitment.Bytes(), ul.LockingParams...)
				if err != nil {
					return err
				}
				dsKey = walletServerNullifierKeyPrefix + serializedViewKey + "/" + nullifier.String()
				if err := dsPutIndexValue(dbtx, idx, dsKey, out.Commitment); err != nil {
					continue
				}
				idx.nullifiers[nullifier] = commitmentWithKey{
					commitment: types.NewID(out.Commitment),
					viewKey:    match.Key,
				}

				if !notifiedKeys[match.Key] {
					go func() {
						idx.subMtx.RLock()
						for _, sub := range idx.subs {
							sub.C <- &UserTransaction{
								Tx:      tx,
								ViewKey: match.Key,
							}
						}
						idx.subMtx.RUnlock()
					}()
					notifiedKeys[match.Key] = true
				}
			} else {
				idx.acc.Insert(out.Commitment, false)
			}
		}
		for _, n := range tx.Nullifiers() {
			if cwk, ok := idx.nullifiers[n]; ok {
				viewKey, err := crypto.MarshalPrivateKey(cwk.viewKey)
				if err != nil {
					continue
				}
				serializedViewKey := hex.EncodeToString(viewKey)
				dsKey := walletServerTxKeyPrefix + serializedViewKey + "/" + tx.ID().String()
				if err := dsPutIndexValue(dbtx, idx, dsKey, nil); err != nil {
					continue
				}

				dsKey = walletServerNullifierKeyPrefix + serializedViewKey + "/" + n.String()
				if err := dsDeleteIndexValue(dbtx, idx, dsKey); err != nil {
					continue
				}

				idx.acc.DropProof(cwk.commitment.Bytes())
				delete(idx.nullifiers, n)

				if !notifiedKeys[cwk.viewKey] {
					go func() {
						idx.subMtx.RLock()
						for _, sub := range idx.subs {
							sub.C <- &UserTransaction{
								Tx:      tx,
								ViewKey: cwk.viewKey,
							}
						}
						idx.subMtx.RUnlock()
					}()
					notifiedKeys[cwk.viewKey] = true
				}
			}
		}
	}
	idx.bestBlockID = blk.ID()
	idx.bestBlockHeight = blk.Header.Height
	return nil
}

// GetTransactionsIDs returns the transaction IDs stored for the given viewKey
func (idx *WalletServerIndex) GetTransactionsIDs(ds repo.Datastore, viewKey crypto.PrivKey) ([]types.ID, error) {
	if _, ok := viewKey.(*icrypto.Curve25519PrivateKey); !ok {
		return nil, errors.New("viewKey is not curve25519 private key")
	}
	dbtx, err := ds.NewTransaction(context.Background(), false)
	if err != nil {
		return nil, err
	}

	key, err := crypto.MarshalPrivateKey(viewKey)
	if err != nil {
		return nil, err
	}

	_, err = dsFetchIndexValue(ds, idx, walletServerViewKeyPrefix+hex.EncodeToString(key))
	if err != nil && err != datastore.ErrNotFound {
		return nil, err
	} else if err == datastore.ErrNotFound {
		return nil, errors.New("view key not registered")
	}

	query, err := dsPrefixQueryIndexValue(dbtx, &WalletServerIndex{}, walletServerTxKeyPrefix+hex.EncodeToString(key))
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

	timeBytes, err := time.Now().MarshalBinary()
	if err != nil {
		return nil, err
	}

	if err := dsPutIndexValue(dbtx, idx, walletServerViewKeyPrefix+hex.EncodeToString(key), timeBytes); err != nil {
		return nil, err
	}

	return txids, dbtx.Commit(context.Background())
}

// GetTxoProofs returns the txo inclusion proof and merkle root for the provided commitments
func (idx *WalletServerIndex) GetTxoProofs(commitments []types.ID) ([]*blockchain.InclusionProof, types.ID, error) {
	idx.stateMtx.RLock()
	defer idx.stateMtx.RUnlock()

	proofs := make([]*blockchain.InclusionProof, 0, len(commitments))
	for _, commitment := range commitments {
		proof, err := idx.acc.GetProof(commitment.Bytes())
		if err != nil {
			return nil, types.ID{}, err
		}
		proofs = append(proofs, proof)
	}
	return proofs, idx.acc.Root(), nil
}

// Close closes the wallet server index
func (idx *WalletServerIndex) Close(ds repo.Datastore) error {
	close(idx.quit)

	return idx.flush(ds)
}

// RegisterViewKey registers a new user with the index. It will track transactions for this user.
func (idx *WalletServerIndex) RegisterViewKey(ds repo.Datastore, viewKey crypto.PrivKey, serializedLockingScript []byte) error {
	if _, ok := viewKey.(*icrypto.Curve25519PrivateKey); !ok {
		return errors.New("viewKey is not curve25519 private key")
	}

	ser, err := crypto.MarshalPrivateKey(viewKey)
	if err != nil {
		return err
	}
	idx.scanner.AddKeys(viewKey.(*icrypto.Curve25519PrivateKey))

	dbtx, err := ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}

	timeBytes, err := time.Now().MarshalBinary()
	if err != nil {
		return err
	}

	if err := dsPutIndexValue(dbtx, idx, walletServerViewKeyPrefix+hex.EncodeToString(ser), timeBytes); err != nil {
		return err
	}

	if err := dsPutIndexValue(dbtx, idx, walletServerLockingScriptPrefix+hex.EncodeToString(ser), serializedLockingScript); err != nil {
		return err
	}

	return dbtx.Commit(context.Background())
}

// RescanViewkey loads historical blocks from disk from the provided checkpoint (or genesis if checkpoint is nil)
// and scans them looking for transactions for the provided viewKey. If transactions are found the internal state
// is updated.
func (idx *WalletServerIndex) RescanViewkey(ds repo.Datastore, viewKey crypto.PrivKey, accumulatorCheckpoint *blockchain.Accumulator, checkpointHeight uint32, getBlockFunc func(uint32) (*blocks.Block, error)) error {
	if _, ok := viewKey.(*icrypto.Curve25519PrivateKey); !ok {
		return errors.New("viewKey is not curve25519 private key")
	}
	acc := accumulatorCheckpoint
	if checkpointHeight == 0 {
		acc = blockchain.NewAccumulator()
		genesisBlk, err := getBlockFunc(0)
		if err != nil {
			log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
			return err
		}
		for _, out := range genesisBlk.Outputs() {
			acc.Insert(out.Commitment, false)
		}
	}
	if acc == nil {
		errStr := "accumulator checkpoint is nil"
		log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", errStr))
		return errors.New(errStr)
	}

	// Let's grab the current height of the chain to avoid repeated
	// locks during the following loop.
	idx.stateMtx.RLock()
	bestHeight := idx.bestBlockHeight
	idx.stateMtx.RUnlock()

	scanner := walletlib.NewTransactionScanner(viewKey.(*icrypto.Curve25519PrivateKey))
	height := checkpointHeight + 1
	nullifiers := make(map[types.Nullifier]commitmentWithKey)
	for {
		blk, err := getBlockFunc(height)
		if err != nil {
			log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
			return err
		}
		matches := scanner.ScanOutputs(blk)
		for _, tx := range blk.Transactions {
			for _, out := range tx.Outputs() {
				match, ok := matches[types.NewID(out.Commitment)]
				if ok {
					dbtx, err := ds.NewTransaction(context.Background(), false)
					if err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					commitmentIndex := idx.acc.NumElements()
					viewKey, err := crypto.MarshalPrivateKey(match.Key)
					if err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					serializedViewKey := hex.EncodeToString(viewKey)
					dsKey := walletServerTxKeyPrefix + serializedViewKey + "/" + tx.ID().String()
					if err := dsPutIndexValue(dbtx, idx, dsKey, nil); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}

					var note types.SpendNote
					if err := note.Deserialize(match.DecryptedNote); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}

					serializedLockingScript, err := dsFetchIndexValueWithTx(dbtx, idx, walletServerLockingScriptPrefix+hex.EncodeToString(viewKey))
					if err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					ul := new(types.LockingScript)
					if err := ul.Deserialize(serializedLockingScript); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					scriptHash, err := ul.Hash()
					if err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					if scriptHash.Compare(note.ScriptHash) != 0 {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}

					nullifier, err := types.CalculateNullifier(commitmentIndex, note.Salt, ul.ScriptCommitment.Bytes(), ul.LockingParams...)
					if err != nil {
						return err
					}
					dsKey = walletServerNullifierKeyPrefix + serializedViewKey + "/" + nullifier.String()
					if err := dsPutIndexValue(dbtx, idx, dsKey, out.Commitment); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					nullifiers[nullifier] = commitmentWithKey{
						commitment: types.NewID(out.Commitment),
						viewKey:    match.Key,
					}
					if err := dbtx.Commit(context.Background()); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
				}
				acc.Insert(out.Commitment, ok)
			}
			for _, n := range tx.Nullifiers() {
				if cwk, ok := nullifiers[n]; ok {
					dbtx, err := ds.NewTransaction(context.Background(), false)
					if err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					viewKey, err := crypto.MarshalPrivateKey(cwk.viewKey)
					if err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					serializedViewKey := hex.EncodeToString(viewKey)
					dsKey := walletServerTxKeyPrefix + serializedViewKey + "/" + tx.ID().String()
					if err := dsPutIndexValue(dbtx, idx, dsKey, nil); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}

					dsKey = walletServerNullifierKeyPrefix + serializedViewKey + "/" + n.String()
					if err := dsDeleteIndexValue(dbtx, idx, dsKey); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}

					if err := dbtx.Commit(context.Background()); err != nil {
						log.WithCaller(true).Error("Wallet server index error rescanning chain", log.Args("error", err))
						return err
					}
					acc.DropProof(cwk.commitment.Bytes())
					delete(nullifiers, n)
				}
			}
		}

		// It's likely the chain has moved forward since we last checked
		// the height so let's check again.
		if blk.Header.Height >= bestHeight {
			idx.stateMtx.Lock()
			if blk.Header.Height == idx.bestBlockHeight {
				idx.acc.MergeProofs(acc)
				for k, v := range nullifiers {
					idx.nullifiers[k] = v
				}
				idx.stateMtx.Unlock()
				return nil
			}
			bestHeight = idx.bestBlockHeight
			idx.stateMtx.Unlock()
		}
		height++
	}
}

// Subscribe returns a subscription to the stream of user transactions.
func (idx *WalletServerIndex) Subscribe() *Subscription {
	sub := &Subscription{
		C:  make(chan *UserTransaction),
		id: rand.Uint64(),
	}
	sub.Close = func() {
		idx.subMtx.Lock()
		go func() {
			for range sub.C {
			}
		}()
		close(sub.C)
		delete(idx.subs, sub.id)
		idx.subMtx.Unlock()
	}
	idx.subMtx.Lock()
	idx.subs[sub.id] = sub
	idx.subMtx.Unlock()
	return sub
}

func (idx *WalletServerIndex) run(ds repo.Datastore) {
	staleUserTicker := time.NewTicker(staleUserTickerInterval)
	flushTicker := time.NewTicker(flushTickerInterval)
	for {
		select {
		case <-staleUserTicker.C:
			dbtx, err := ds.NewTransaction(context.Background(), true)
			if err != nil {
				log.WithCaller(true).Error("Wallet server index error deleting stale users", log.Args("error", err))
				continue
			}
			query, err := dsPrefixQueryIndexValue(dbtx, idx, walletServerViewKeyPrefix)
			if err != nil {
				log.WithCaller(true).Error("Wallet server index error deleting stale users", log.Args("error", err))
				continue
			}

			for r := range query.Next() {
				var lastSeen time.Time
				if err := lastSeen.UnmarshalBinary(r.Value); err != nil {
					log.WithCaller(true).Error("Wallet server index error deleting stale users", log.Args("error", err))
					continue
				}
				if time.Now().After(lastSeen.Add(staleUserThreshold)) {
					s := strings.Split(r.Key, "/")
					keyStr := s[len(s)-1]
					keyBytes, err := hex.DecodeString(keyStr)
					if err != nil {
						continue
					}
					viewKey, err := crypto.UnmarshalPrivateKey(keyBytes)
					if err != nil {
						continue
					}
					if _, ok := viewKey.(*icrypto.Curve25519PrivateKey); !ok {
						continue
					}

					idx.stateMtx.Lock()
					idx.scanner.RemoveKey(viewKey.(*icrypto.Curve25519PrivateKey))
					idx.stateMtx.Unlock()

					if err := dsDeleteIndexValue(dbtx, idx, r.Key); err != nil {
						log.WithCaller(true).Error("Wallet server index error deleting stale users", log.Args("error", err))
						continue
					}

					dsKey := walletServerLockingScriptPrefix + keyStr
					if err := dsDeleteIndexValue(dbtx, idx, dsKey); err != nil {
						log.WithCaller(true).Error("Wallet server index error deleting stale users", log.Args("error", err))
						continue
					}
				}

			}
			if err := dbtx.Commit(context.Background()); err != nil {
				log.WithCaller(true).Error("Wallet server index error deleting stale users", log.Args("error", err))
			}

		case <-flushTicker.C:
			if err := idx.flush(ds); err != nil {
				log.WithCaller(true).Error("Wallet server index error flushing to disk", log.Args("error", err))
			}
		case <-idx.quit:
			return
		}
	}
}

func (idx *WalletServerIndex) flush(ds repo.Datastore) error {
	idx.stateMtx.Lock()
	defer idx.stateMtx.Unlock()

	ser, err := blockchain.SerializeAccumulator(idx.acc)
	if err != nil {
		return err
	}

	heightBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(heightBytes, idx.bestBlockHeight)

	dbtx, err := ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	if err := dsPutIndexValue(dbtx, idx, walletServerAccumulatorKey, ser); err != nil {
		return err
	}
	if err := dsPutIndexValue(dbtx, idx, walletServerBestBlockKey, append(heightBytes, idx.bestBlockID.Bytes()...)); err != nil {
		return err
	}
	if err := dsPutIndexerHeight(dbtx, idx, idx.bestBlockHeight); err != nil {
		return err
	}
	return dbtx.Commit(context.Background())
}

// DropWalletServerIndex deletes the wallet server index from the datastore
func DropWalletServerIndex(ds repo.Datastore) error {
	return dsDropIndex(ds, &WalletServerIndex{})
}
