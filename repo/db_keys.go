// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

const (
	// NetworkKeyDatastoreKey is the datastore key for the network (libp2p) private key.
	NetworkKeyDatastoreKey = "/ilxd/libp2pkey/"
	// ValidatorDatastoreKeyPrefix is the datastore key prefix for the validators.
	ValidatorDatastoreKeyPrefix = "/ilxd/validator/"
	// ValidatorSetLastFlushHeight is the datastore key for last flush height of the validator set.
	ValidatorSetLastFlushHeight = "/ilxd/validatorsetlastflushheight/"
	// ValidatorSetConsistencyStatusKey is the datastore key for the validator set flush state.
	ValidatorSetConsistencyStatusKey = "/ilxd/validatorsetconsistencystatus/"
	// AccumulatorConsistencyStatusKey is the datastore key for the accumulator flush state.
	AccumulatorConsistencyStatusKey = "/ilxd/accumulatorconsistencystatus/"
	// AccumulatorLastFlushHeight is the datastore key for last flush height of the accumulator.
	AccumulatorLastFlushHeight = "/ilxd/accumulatorlastflushheight/"
	// BlockByHeightKeyPrefix is the datastore key prefix for mapping block heights to block IDs.
	BlockByHeightKeyPrefix = "/ilxd/blockbyheight/"
	// BlockKeyPrefix is the datastore key prefix for storing block headers by blockID.
	BlockKeyPrefix = "/ilxd/block/"
	// BlockIndexStateKey is the datastore key used to store the block index best state.
	BlockIndexStateKey = "/ilxd/blockindex/"
	// NullifierKeyPrefix is the datastore key prefix for storing nullifiers in the nullifier set.
	NullifierKeyPrefix = "/ilxd/nullifier/"
	// TxoRootKeyPrefix is the datastore key prefix for storing a txo root in the database.
	TxoRootKeyPrefix = "/ilxd/txoroot/"
	// TreasuryBalanceKey is the datastire key for storing the balance of the treasury in the database.
	TreasuryBalanceKey = "/ilxd/treasury/"
	// AccumulatorStateKey is the datastore key for storing the accumulator state.
	AccumulatorStateKey = "/ilxd/accumulator/"
	// AccumulatorCheckpointKey is the datastore key for storing accumulator checkpoints.
	AccumulatorCheckpointKey = "/ilxd/accumulatorcheckpoint/"
	// CoinSupplyKey is the datastore key for storing the current supply of coins.
	CoinSupplyKey = "/ilxd/coinsupply/"
	// IndexerHeightKeyPrefix is the datastore key prefix for mapping indexers to sync heights.
	IndexerHeightKeyPrefix = "/ilxd/indexerheight/"
	// IndexKeyPrefix is the datastore key used by each indexer. This must be extended to use.
	IndexKeyPrefix = "/ilxd/index/"
	// ConnGaterKeyPrefix is the datastore namespace key used by the conngater.
	ConnGaterKeyPrefix = "/ilxd/conngater/"
	// AutostakeDatastoreKey is the datastore key used to store the autostake bool.
	AutostakeDatastoreKey = "/ilxd/autostake/"
	// PrunedBlockchainDatastoreKey is the datastore key used to store a flag setting whether the chain has ever been pruned.
	PrunedBlockchainDatastoreKey = "/ilxd/pruned/"
	// CachedAddrInfoDatastoreKey is the datastore key used to persist addrinfos from the peerstore.
	CachedAddrInfoDatastoreKey = "/ilxd/peerstore/addrinfo/"
	// TreasuryWhitelistDatastoreKeyPrefix is the datastore key prefix for the treasury whitelist.
	TreasuryWhitelistDatastoreKeyPrefix = "/ilxd/whitelist/"

	// TxIndexKey is the datastore key for the transaction index.
	TxIndexKey = "txindex"
	// WalletServerIndexKey is the datastore key for the wallet server index.
	WalletServerIndexKey = "walletserverindex"
	// AddrIndexKey is the datastore key for the address index.
	AddrIndexKey = "addrindex"
	// WalletServerAccumulatorKey is the accumulator key used by the wallet server index.
	WalletServerAccumulatorKey = "accumulator"
	// WalletServerBestBlockKey is the best block key used by the wallet server index.
	WalletServerBestBlockKey = "bestblockid"
	// WalletServerViewKeyPrefix is the view key prefix used by the wallet server index.
	WalletServerViewKeyPrefix = "viewkey/"
	// WalletServerLockingScriptPrefix is the locking script prefix used by the wallet server index.
	WalletServerLockingScriptPrefix = "lockingscript/"
	// WalletServerNullifierKeyPrefix is the nullifier key prefix used by the wallet server index.
	WalletServerNullifierKeyPrefix = "nullifier/"
	// WalletServerTxKeyPrefix is the tx key prefix used by the wallet server index.
	WalletServerTxKeyPrefix = "tx/"
	// AddrIndexAddrKeyPrefix is the address key prefix used by the address index.
	AddrIndexAddrKeyPrefix = "addr/"
	// AddrIndexNulliferKeyPrefix is the nullifier key prefix used by the address index.
	AddrIndexNulliferKeyPrefix = "nullifier/"
	// AddrIndexOutputIndexKey is the outputs datastore key used by the address index.
	AddrIndexOutputIndexKey = "outputs/"
	// AddrIndexMetadataPrefixKey is the metadata prefix datastore key used by the address index.
	AddrIndexMetadataPrefixKey = "metadata/"
	// AddrIndexValidatorTxPrefixKey is the validator transaction datastore key prefix used by the address index.
	AddrIndexValidatorTxPrefixKey = "validatortx/"
)
