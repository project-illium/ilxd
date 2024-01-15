// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/indexers"
	"github.com/project-illium/ilxd/consensus"
	"github.com/project-illium/ilxd/gen"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/net"
	params "github.com/project-illium/ilxd/params"
	policy2 "github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/rpc"
	"github.com/project-illium/ilxd/sync"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/walletlib"
	"github.com/project-illium/walletlib/client"
	"go.uber.org/zap"
	"sort"
	stdsync "sync"
	"time"
)

const (
	maxOrphanDuration     = time.Hour
	maxOrphans            = 100
	orphanResyncThreshold = 5
)

var log = zap.S()

type orphanBlock struct {
	blk          *blocks.Block
	relayingPeer peer.ID
	firstSeen    time.Time
}

// Server is the main class that brings all the constituent parts together
// into a full node.
type Server struct {
	cancelFunc   context.CancelFunc
	ctx          context.Context
	config       *repo.Config
	params       *params.NetworkParams
	ds           repo.Datastore
	network      *net.Network
	blockchain   *blockchain.Blockchain
	mempool      *mempool.Mempool
	engine       *consensus.ConsensusEngine
	chainService *sync.ChainService
	syncManager  *sync.SyncManager
	generator    *gen.BlockGenerator
	grpcServer   *rpc.GrpcServer
	wallet       *walletlib.Wallet
	coinbaseAddr walletlib.Address

	orphanBlocks map[types.ID]*orphanBlock
	orphanLock   stdsync.RWMutex

	activeInventory map[types.ID]*blocks.Block
	inventoryLock   stdsync.RWMutex

	submittedTxs     map[types.ID]struct{}
	submittedTxsLock stdsync.RWMutex

	inflightRequests map[types.ID]bool
	inflightLock     stdsync.RWMutex
	policy           *policy2.Policy
	autoStake        bool
	autoStakeLock    stdsync.RWMutex
	coinbasesToStake map[types.ID]struct{}
	networkKey       crypto.PrivKey

	ready chan struct{}
}

// BuildServer is the constructor for the server. We pass in the config file here
// and use it to configure all the various parts of the Server.
func BuildServer(config *repo.Config) (*Server, error) {
	printSplashScreen()
	ctx, cancel := context.WithCancel(context.Background()) //nolint:govet

	s := Server{ready: make(chan struct{})}
	defer close(s.ready)

	// Logging
	zapLevel, err := setupLogging(config.LogDir, config.LogLevel, config.Testnet)
	if err != nil {
		return nil, err //nolint:govet
	}

	if config.EnableDebugLogging {
		golog.SetDebugLogging()
	}

	// Policy
	policy := policy2.NewPolicy(
		types.Amount(config.Policy.MinFeePerKilobyte),
		types.Amount(config.Policy.MinStake),
		config.Policy.BlocksizeSoftLimit,
	)

	for _, id := range config.Policy.TreasuryWhitelist {
		w, err := types.NewIDFromString(id)
		if err != nil {
			return nil, err
		}
		s.policy.AddToTreasuryWhitelist(w)
	}

	// Parameter selection
	var netParams *params.NetworkParams
	if config.Testnet {
		netParams = &params.Testnet1Params
	} else if config.Alphanet {
		netParams = &params.AlphanetParams
	} else if config.Regtest {
		netParams = &params.RegestParams
		if config.RegtestVal {
			config.WalletSeed = params.RegtestMnemonicSeed
		}
	} else {
		netParams = &params.MainnetParams
	}

	var (
		prover   zk.Prover
		verifier zk.Verifier
	)
	if config.MockProofs {
		if !netParams.AllowMockProofs {
			return nil, errors.New("mock proofs not allowed with network params selection")
		}
		prover = &zk.MockProver{}
		v := &zk.MockVerifier{}
		v.SetValid(true)
		verifier = v
	} else {
		// Load public parameters
		zk.LoadZKPublicParameters()
		prover = &zk.LurkProver{}
		verifier = &zk.LurkVerifier{}
	}

	if config.CoinbaseAddress != "" {
		addr, err := walletlib.DecodeAddress(config.CoinbaseAddress, netParams)
		if err != nil {
			return nil, err
		}
		s.coinbaseAddr = addr
	}

	// Setup up badger datastore
	ds, err := badger.NewDatastore(config.DataDir, &badger.DefaultOptions)
	if err != nil {
		return nil, err
	}

	// Create the blockchain
	sigCache := blockchain.NewSigCache(blockchain.DefaultSigCacheSize)
	proofCache := blockchain.NewProofCache(blockchain.DefaultProofCacheSize)
	var (
		indexerList []indexers.Indexer
		txIndex     *indexers.TxIndex
		wsIndex     *indexers.WalletServerIndex
	)
	if !config.NoTxIndex && !config.DropTxIndex {
		txIndex = indexers.NewTxIndex()
		indexerList = append(indexerList, txIndex)
	}

	if config.WSIndex && !config.DropWSIndex {
		wsIndex, err = indexers.NewWalletServerIndex(ds)
		if err != nil {
			return nil, err
		}
		indexerList = append(indexerList, wsIndex)
	}

	if config.WSIndex && config.NoTxIndex {
		return nil, errors.New("tx index must be used with wallet server index")
	}

	blockchainOpts := []blockchain.Option{
		blockchain.Params(netParams),
		blockchain.Datastore(ds),
		blockchain.MaxNullifiers(blockchain.DefaultMaxNullifiers),
		blockchain.MaxTxoRoots(blockchain.DefaultMaxTxoRoots),
		blockchain.SignatureCache(sigCache),
		blockchain.SnarkProofCache(proofCache),
		blockchain.Verifier(verifier),
	}

	if config.Prune {
		blockchainOpts = append(blockchainOpts, blockchain.Prune())
	}

	if len(indexerList) != 0 {
		indexManager := indexers.NewIndexManager(ds, indexerList)
		blockchainOpts = append(blockchainOpts, blockchain.Indexer(indexManager))
	}

	if config.DropTxIndex {
		if err := indexers.DropTxIndex(ds); err != nil {
			return nil, err
		}
	}
	if config.DropWSIndex {
		if err := indexers.DropWalletServerIndex(ds); err != nil {
			return nil, err
		}
	}

	chain, err := blockchain.NewBlockchain(blockchainOpts...)
	if err != nil {
		return nil, err
	}

	// Create wallet
	walletOpts := []walletlib.Option{
		walletlib.Prover(prover),
		walletlib.DataDir(config.WalletDir),
		walletlib.Params(netParams),
		walletlib.FeePerKB(policy.GetMinFeePerKilobyte()),
		walletlib.BlockchainSource(s.makeBlockchainClient(chain)),
	}
	if config.WalletSeed != "" {
		walletOpts = append(walletOpts, walletlib.MnemonicSeed(config.WalletSeed))
	}
	wallet, err := walletlib.NewWallet(walletOpts...)
	if err != nil {
		return nil, err
	}

	// Load or create the private key for the node
	var privKey crypto.PrivKey
	has, err := repo.HasNetworkKey(ds)
	if err != nil {
		return nil, err
	}
	if has {
		privKey, err = repo.LoadNetworkKey(ds)
		if err != nil {
			return nil, err
		}
	} else {
		if netParams.Name == params.RegestParams.Name && config.RegtestVal {
			privKey, err = crypto.UnmarshalPrivateKey(params.RegtestGenesisKey)
			if err != nil {
				return nil, err
			}
			if err := repo.PutNetworkKey(ds, privKey); err != nil {
				return nil, err
			}
		} else {
			privKey, err = wallet.NetworkKey()
			if err != nil {
				return nil, err
			}
			if err := repo.PutNetworkKey(ds, privKey); err != nil {
				return nil, err
			}
		}
	}

	if config.NetworkKey != "" {
		keyBytes, err := hex.DecodeString(config.NetworkKey)
		if err != nil {
			return nil, err
		}
		privKey, err = crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}
	}

	// Select seed addresses
	var seedAddrs []string
	if config.SeedAddrs != nil {
		seedAddrs = config.SeedAddrs
	} else {
		seedAddrs = netParams.SeedAddrs
	}

	// Select listen addresses
	var listenAddrs []string
	if config.ListenAddrs != nil {
		listenAddrs = config.ListenAddrs
	} else {
		listenAddrs = netParams.ListenAddrs
	}

	// Mempool
	mempoolOpts := []mempool.Option{
		mempool.SignatureCache(sigCache),
		mempool.ProofCache(proofCache),
		mempool.Params(netParams),
		mempool.BlockchainView(chain),
		mempool.MinStake(policy.GetMinStake()),
		mempool.FeePerKilobyte(policy.GetMinFeePerKilobyte()),
		mempool.Verifier(verifier),
	}

	mpool, err := mempool.NewMempool(mempoolOpts...)
	if err != nil {
		return nil, err
	}

	// Network
	networkOpts := []net.Option{
		net.Datastore(ds),
		net.SeedAddrs(seedAddrs),
		net.ListenAddrs(listenAddrs),
		net.UserAgent(config.UserAgent),
		net.PrivateKey(privKey),
		net.Params(netParams),
		net.BlockValidator(s.handleIncomingBlock),
		net.MempoolValidator(s.processMempoolTransaction),
		net.MaxBanscore(config.MaxBanscore),
		net.BanDuration(config.BanDuration),
		net.MaxMessageSize(config.Policy.MaxMessageSize),
	}
	if config.DisableNATPortMap {
		networkOpts = append(networkOpts, net.DisableNatPortMap())
	}
	hostID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	if chain.ValidatorExists(hostID) {
		networkOpts = append(networkOpts, net.ForceDHTServerMode())
	}

	network, err := net.NewNetwork(ctx, networkOpts...)
	if err != nil {
		return nil, err
	}

	valConn := net.NewValidatorConnector(network.Host(), hostID, chain.GetValidator, chain.Validators, chain.Subscribe)

	engine, err := consensus.NewConsensusEngine(ctx, []consensus.Option{
		consensus.Params(netParams),
		consensus.Network(network),
		consensus.ValidatorConnector(valConn),
		consensus.Chooser(chain),
		consensus.RequestBlock(s.requestBlock),
		consensus.GetBlockID(chain.GetBlockIDByHeight),
		consensus.PeerID(network.Host().ID()),
	}...)
	if err != nil {
		return nil, err
	}

	grpcServer, err := newGrpcServer(config.RPCOpts, &rpc.GrpcServerConfig{
		Chain:                chain,
		Network:              network,
		Policy:               policy,
		Wallet:               wallet,
		Prover:               prover,
		BroadcastTxFunc:      s.submitTransaction,
		SetLogLevelFunc:      zapLevel.SetLevel,
		ReindexChainFunc:     s.reIndexChain,
		RequestBlockFunc:     s.requestBlock,
		AutoStakeFunc:        s.setAutostake,
		NetworkKeyFunc:       s.getNetworkKey,
		ChainParams:          netParams,
		Ds:                   ds,
		TxMemPool:            mpool,
		TxIndex:              txIndex,
		WSIndex:              wsIndex,
		DisableNodeService:   config.RPCOpts.DisableNodeService,
		DisableWalletService: config.RPCOpts.DisableWalletService,
		DisableWalletServer:  config.RPCOpts.DisableWalletServerService || wsIndex == nil,
	})
	if err != nil {
		return nil, err
	}

	generator, err := gen.NewBlockGenerator([]gen.Option{
		gen.Blockchain(chain),
		gen.PrivateKey(privKey),
		gen.Mempool(mpool),
		gen.BroadcastFunc(network.BroadcastBlock),
	}...)
	if err != nil {
		return nil, err
	}

	autostake, err := ds.Get(context.Background(), datastore.NewKey(repo.AutostakeDatastoreKey))
	if err != nil && !errors.Is(datastore.ErrNotFound, err) {
		return nil, err
	}

	s.chainService, err = sync.NewChainService(ctx, s.fetchBlock, chain, network, netParams)
	if err != nil {
		return nil, err
	}

	s.ctx = ctx
	s.cancelFunc = cancel
	s.config = config
	s.params = netParams
	s.ds = ds
	s.network = network
	s.mempool = mpool
	s.blockchain = chain
	s.engine = engine
	s.syncManager = sync.NewSyncManager(&sync.SyncManagerConfig{
		Ctx:               ctx,
		Chain:             chain,
		Network:           network,
		Params:            netParams,
		CS:                s.chainService,
		Chooser:           s.consensusChoose,
		IsCurrentCallback: s.handleCurrentStatusChange,
		ProofCache:        proofCache,
		SigCache:          sigCache,
		Verifier:          verifier,
	})
	s.orphanBlocks = make(map[types.ID]*orphanBlock)
	s.activeInventory = make(map[types.ID]*blocks.Block)
	s.submittedTxs = make(map[types.ID]struct{})
	s.inflightRequests = make(map[types.ID]bool)
	s.orphanLock = stdsync.RWMutex{}
	s.inventoryLock = stdsync.RWMutex{}
	s.inflightLock = stdsync.RWMutex{}
	s.autoStakeLock = stdsync.RWMutex{}
	s.submittedTxsLock = stdsync.RWMutex{}
	s.policy = policy
	s.generator = generator
	s.grpcServer = grpcServer
	s.wallet = wallet
	s.autoStake = bytes.Equal(autostake, []byte{0x01})
	s.coinbasesToStake = make(map[types.ID]struct{})
	s.networkKey = privKey

	chain.Subscribe(s.handleBlockchainNotification)

	s.printListenAddrs()

	s.wallet.Start()

	go s.syncManager.Start()

	// If we are the genesis validator then start generating immediately.
	_, height, _ := chain.BestBlock()
	if height == 0 {
		if chain.Validators()[0].PeerID == network.Host().ID() {
			s.syncManager.SetCurrent()
			generator.Start()
		}
	}

	return &s, nil
}

func (s *Server) processMempoolTransaction(tx *transactions.Transaction) error {
	<-s.ready

	// We will let our own txs through even if we're not current.
	s.submittedTxsLock.RLock()
	_, ok := s.submittedTxs[tx.ID()]
	s.submittedTxsLock.RUnlock()

	if !s.syncManager.IsCurrent() && !ok {
		return blockchain.NotCurrentError("chain not current")
	}

	s.submittedTxsLock.Lock()
	delete(s.submittedTxs, tx.ID())
	s.submittedTxsLock.Unlock()
	return s.mempool.ProcessTransaction(tx)
}

func (s *Server) submitTransaction(tx *transactions.Transaction) error {
	<-s.ready

	s.submittedTxsLock.Lock()
	s.submittedTxs[tx.ID()] = struct{}{}
	s.submittedTxsLock.Unlock()

	// Pubsub has the mempool validation handler register on it, so function
	// will submit it to the mempool, validate it, and return an error if
	// validation fails.
	return s.network.BroadcastTransaction(tx)
}

func (s *Server) getNetworkKey() (crypto.PrivKey, error) {
	return repo.LoadNetworkKey(s.ds)
}

func (s *Server) handleIncomingBlock(xThinnerBlk *blocks.XThinnerBlock, p peer.ID) error {
	<-s.ready
	_, height, _ := s.blockchain.BestBlock()

	if !s.syncManager.IsCurrent() && xThinnerBlk.Header.Height != height+1 {
		return blockchain.NotCurrentError("chain not current")
	}

	// Try to decode the block. This should succeed most of the time unless
	// the merkle root is invalid.
	blockID := xThinnerBlk.ID()
	s.inflightLock.Lock()
	s.inflightRequests[blockID] = true
	s.inflightLock.Unlock()

	blk, err := s.decodeXthinner(xThinnerBlk, p)
	if err != nil {
		return err
	}

	time.AfterFunc(time.Minute*5, func() {
		s.inflightLock.Lock()
		delete(s.inflightRequests, blockID)
		s.inflightLock.Unlock()
	})

	return s.processBlock(blk, p, false)
}

func (s *Server) handleBlockchainNotification(ntf *blockchain.Notification) {
	<-s.ready

	switch ntf.Type {
	case blockchain.NTBlockConnected:
		if blk, ok := ntf.Data.(*blocks.Block); ok {
			s.mempool.RemoveBlockTransactions(blk.Transactions)

			s.autoStakeLock.RLock()
			toStake := s.coinbasesToStake
			s.autoStakeLock.RUnlock()
			if len(toStake) > 0 {
				for txid := range toStake {
					for _, tx := range blk.Transactions {
						if tx.ID() == txid {
							if err := s.wallet.Stake([]types.ID{types.NewID(tx.GetCoinbaseTransaction().Outputs[0].Commitment)}); err != nil {
								log.Errorf("Error autostaking coinbase: %s", err)
								continue
							}
							s.autoStakeLock.Lock()
							delete(s.coinbasesToStake, txid)
							s.autoStakeLock.Unlock()
						}
					}
				}
			}
		}
	case blockchain.NTAddValidator:
		if pid, ok := ntf.Data.(peer.ID); ok {
			if pid == s.network.Host().ID() {
				if !s.generator.Active() {
					s.generator.Start()
				}
			} else {
				s.network.ConnManager().Protect(pid, net.ValidatorProtectionFlag)
			}
		}
	case blockchain.NTRemoveValidator:
		if pid, ok := ntf.Data.(peer.ID); ok {
			if pid == s.network.Host().ID() {
				if s.generator.Active() {
					s.generator.Close()
				}
			} else {
				s.network.ConnManager().Unprotect(pid, net.ValidatorProtectionFlag)
			}
		}
	case blockchain.NTNewEpoch:
		log.Info("New blockchain epoch")
		validator, err := s.blockchain.GetValidator(s.network.Host().ID())
		if err == nil && validator.UnclaimedCoins > 0 {
			tx, err := s.wallet.BuildCoinbaseTransaction(validator.UnclaimedCoins, s.coinbaseAddr, s.networkKey)
			if err != nil {
				log.Errorf("Error building auto coinbase transaction: %s", err)
				return
			}
			if err := s.submitTransaction(tx); err != nil {
				log.Errorf("Error submitting auto coinbase transaction: %s", err)
				return
			}

			s.autoStakeLock.RLock()
			defer s.autoStakeLock.RUnlock()
			if s.autoStake {
				s.coinbasesToStake[tx.ID()] = struct{}{}
			}
		}
	}
}

func (s *Server) setAutostake(autostake bool) error {
	s.autoStakeLock.Lock()
	defer s.autoStakeLock.Unlock()

	b := []byte{0x00}
	if autostake {
		b = []byte{0x01}
	}
	s.autoStake = autostake
	return s.ds.Put(context.Background(), datastore.NewKey(repo.AutostakeDatastoreKey), b)
}

func (s *Server) processBlock(blk *blocks.Block, relayingPeer peer.ID, recheck bool) error {
	<-s.ready
	err := s.blockchain.CheckConnectBlock(blk)

	switch err.(type) {
	case blockchain.OrphanBlockError:
		// An orphan is a block's whose height is greater than
		// our current blockchain tip. It might be valid, but
		// we can't validate it until we connect the parent.
		// We'll store it in memory and will circle back after
		// we connect the next block.
		s.orphanLock.Lock()
		s.limitOrphans()
		s.orphanBlocks[blk.ID()] = &orphanBlock{
			blk:          blk,
			firstSeen:    time.Now(),
			relayingPeer: relayingPeer,
		}

		// This really shouldn't happen but if we're piling up the orphans
		// and we haven't connected a block in a little bit let's trigger
		// a resync since we likely missed a block.
		_, _, tipTimstamp := s.blockchain.BestBlock()
		if len(s.orphanBlocks) >= orphanResyncThreshold &&
			s.syncManager.IsCurrent() &&
			time.Now().After(tipTimstamp.Add(time.Minute*5)) {

			s.generator.Close()
			s.syncManager.Close()
			s.syncManager.Start()
		}
		s.orphanLock.Unlock()
		return err
	case blockchain.RuleError:
		if recheck {
			s.network.IncreaseBanscore(relayingPeer, 34, 0)
			return err
		}
		// If the merkle root is invalid it either means we had a collision in the
		// mempool or this block is genuinely invalid.
		//
		// Let's download the txid list from the peer and figure out which it is.
		if blockchain.ErrorIs(err, blockchain.ErrInvalidTxRoot) {
			blk, err := s.fetchBlockTxids(blk, relayingPeer)
			if err != nil {
				s.network.IncreaseBanscore(relayingPeer, 34, 0)

				for _, pid := range s.network.Host().Network().Peers() {
					blk, err = s.fetchBlockTxids(blk, pid)
					if err == nil {
						return s.processBlock(blk, relayingPeer, true)
					}
				}
			} else {
				return s.processBlock(blk, relayingPeer, true)
			}
		} else if blockchain.ErrorIs(err, blockchain.ErrDoesNotConnect) {
			// Small chance of a race condition where we receive a block
			// right after we finalize a block at the same height. We'll
			// only lightly increase the penalty for this to prevent banning
			// nodes for innocent behavior.
			s.network.IncreaseBanscore(relayingPeer, 0, 1)
		} else {
			// Ban nodes that send us invalid blocks.
			s.network.IncreaseBanscore(relayingPeer, 101, 0)
		}
		return err
	}
	if err != nil {
		return err
	}

	callback := make(chan consensus.Status)

	startTime := time.Now()

	isAcceptable, err := s.policy.IsAcceptableBlock(blk)
	if err != nil {
		log.Warnf("Error calculating policy preference: %s", err)
	}

	s.inventoryLock.Lock()
	for _, inv := range s.activeInventory {
		if inv.Header.Height == blk.Header.Height &&
			inv.ID() != blk.ID() &&
			bytes.Equal(inv.Header.Producer_ID, blk.Header.Producer_ID) &&
			time.Unix(blk.Header.Timestamp, 0).Before(time.Unix(inv.Header.Timestamp, 0).Add(gen.MinAllowableTimeBetweenDupBlocks)) {

			// The block producer sent us two blocks at the same height
			// too close together.
			s.network.IncreaseBanscore(relayingPeer, 101, 0)
			s.inventoryLock.Unlock()
			return errors.New("multiple blocks from the same validator")
		}
	}
	s.activeInventory[blk.ID()] = blk
	s.inventoryLock.Unlock()

	s.orphanLock.Lock()
	delete(s.orphanBlocks, blk.ID())
	s.orphanLock.Unlock()

	s.generator.Interrupt(blk.Header.Height)
	log.Debugf("[CONSENSUS] new block: %s", blk.ID())
	s.engine.NewBlock(blk.Header, isAcceptable, callback)

	go func(b *blocks.Block, t time.Time) {
		select {
		case status := <-callback:
			switch status {
			case consensus.StatusFinalized:
				blockID := blk.ID()
				log.Debugf("Block %s finalized in %d milliseconds", blockID, time.Since(t).Milliseconds())
				if err := s.blockchain.ConnectBlock(b, blockchain.BFNone); err != nil {
					log.Warnf("Connect block error: block %s: %s", blockID, err)
				} else {
					log.Infof("New block: %s, (height: %d, transactions: %d)", blockID, blk.Header.Height, len(b.Transactions))
					s.syncManager.SetCurrent()
				}
			case consensus.StatusRejected:
				log.Debugf("Block %s rejected by consensus", b.ID())
			}

			// Leave it here for a little in case a peer requests it.
			// This likely would only happen in a race condition.
			time.AfterFunc(time.Minute, func() {
				s.inventoryLock.Lock()
				delete(s.activeInventory, blk.ID())
				s.inventoryLock.Unlock()
			})

			s.orphanLock.Lock()
			for _, orphan := range s.orphanBlocks {
				if orphan.blk.Header.Height == blk.Header.Height {
					delete(s.orphanBlocks, orphan.blk.ID())
				} else if orphan.blk.Header.Height == blk.Header.Height+1 {
					log.Debugf("Re-procssing orphan at height %d: %s", orphan.blk.Header.Height, orphan.blk.ID())
					go s.processBlock(orphan.blk, orphan.relayingPeer, false)
					break
				} else if time.Since(orphan.firstSeen) > maxOrphanDuration {
					delete(s.orphanBlocks, orphan.blk.ID())
				}
			}
			s.orphanLock.Unlock()
		case <-s.ctx.Done():
			return
		}
	}(blk, startTime)
	return nil
}

func (s *Server) decodeXthinner(xThinnerBlk *blocks.XThinnerBlock, relayingPeer peer.ID) (*blocks.Block, error) {
	<-s.ready
	blk, missing := s.mempool.DecodeXthinner(xThinnerBlk)
	if len(missing) > 0 {
		txs, err := s.chainService.GetBlockTxs(relayingPeer, xThinnerBlk.ID(), missing)
		if err == nil {
			for i, tx := range txs {
				blk.Transactions[missing[i]] = tx
			}
			return blk, nil
		} else {
			s.network.IncreaseBanscore(relayingPeer, 34, 0)
		}

		for _, pid := range s.network.Host().Network().Peers() {
			txs, err := s.chainService.GetBlockTxs(pid, xThinnerBlk.ID(), missing)
			if err == nil {
				for i, tx := range txs {
					blk.Transactions[missing[i]] = tx
				}
				return blk, nil
			}
			// We won't increase the ban score for these peers as they didn't send
			// us the block. If the block is invalid they may not be able to legitimately
			// respond to our request.
		}
		return nil, errors.New("failed to decode from all peers")
	}
	return blk, nil
}

func (s *Server) fetchBlockTxids(blk *blocks.Block, p peer.ID) (*blocks.Block, error) {
	<-s.ready
	txids, err := s.chainService.GetBlockTxids(p, blk.ID())
	if err != nil {
		return nil, err
	}
	if len(txids) != len(blk.Transactions) {
		return nil, errors.New("getblocktxids: peer returned unexpected  number of IDs")
	}
	missing := make([]uint32, 0, len(blk.Transactions))
	for i, tx := range blk.Transactions {
		if tx.ID() != txids[i] {
			missing = append(missing, uint32(i))
		}
	}
	// This function is only called if the block's merkle root was invalid. That
	// can only happen with an otherwise valid block if we had a prefix collision
	// in the mempool. If the missing slice is empty it means that there was no
	// prefix collision detected and the block is genuinely invalid.
	if len(missing) == 0 {
		return nil, errors.New("block invalid merkle root")
	}
	txs, err := s.chainService.GetBlockTxs(p, blk.ID(), missing)
	if err != nil {
		return nil, err
	}
	for i, tx := range txs {
		blk.Transactions[missing[i]] = tx
	}
	return blk, nil
}

func (s *Server) fetchBlock(blockID types.ID) (*blocks.Block, error) {
	<-s.ready
	s.inventoryLock.RLock()
	defer s.inventoryLock.RUnlock()

	if blk, ok := s.activeInventory[blockID]; ok {
		return blk, nil
	}

	return s.blockchain.GetBlockByID(blockID)
}

func (s *Server) consensusChoose(blks []*blocks.Block) (types.ID, error) {
	<-s.ready
	height := blks[0].Header.Height
	for _, blk := range blks {
		if blk.Header.Height != height {
			return types.ID{}, errors.New("not all blocks at the same height")
		}
	}

	respCh := make(chan *types.ID)
	wg := stdsync.WaitGroup{}
	wg.Add(len(blks))
	go func() {
		for _, blk := range blks {
			go func(b *blocks.Block, w *stdsync.WaitGroup) {
				callback := make(chan consensus.Status)
				s.engine.NewBlock(b.Header, false, callback)
				select {
				case status := <-callback:
					if status == consensus.StatusFinalized {
						id := b.Header.ID()
						respCh <- &id
					}
				case <-time.After(time.Second * 30):
				}
				w.Done()
			}(blk, &wg)
		}
		wg.Wait()
		close(respCh)
	}()
	id := <-respCh
	if id != nil {
		return *id, nil
	}
	return types.ID{}, errors.New("no blocks finalized")
}

func (s *Server) requestBlock(blockID types.ID, remotePeer peer.ID) {
	<-s.ready
	if !s.syncManager.IsCurrent() {
		return
	}
	s.inflightLock.RLock()
	if _, ok := s.inflightRequests[blockID]; ok {
		s.inflightLock.RUnlock()
		return
	}
	s.inflightLock.RUnlock()

	s.inflightLock.Lock()
	s.inflightRequests[blockID] = true
	s.inflightLock.Unlock()

	log.Debugf("Requesting unknown block %s from peer %s", blockID, remotePeer.String())
	blk, err := s.chainService.GetBlock(remotePeer, blockID)
	if err != nil {
		s.network.IncreaseBanscore(remotePeer, 0, 30)
		s.inflightLock.Lock()
		delete(s.inflightRequests, blockID)
		s.inflightLock.Unlock()
		return
	}

	s.processBlock(blk, remotePeer, false)

	time.AfterFunc(time.Minute*5, func() {
		s.inflightLock.Lock()
		delete(s.inflightRequests, blockID)
		s.inflightLock.Unlock()
	})
}

func (s *Server) reIndexChain() error {
	<-s.ready
	s.generator.Close()
	s.syncManager.Close()
	if err := s.blockchain.ReindexChainState(); err != nil {
		return err
	}
	s.syncManager.Start()
	return nil
}

func (s *Server) handleCurrentStatusChange() {
	<-s.ready
	if s.blockchain.ValidatorExists(s.network.Host().ID()) {
		s.generator.Start()
	}
}

// Close shuts down all the parts of the server and blocks until
// they finish closing.
func (s *Server) Close() error {
	<-s.ready
	s.cancelFunc()
	s.generator.Close()
	s.syncManager.Close()
	s.engine.Close()
	s.mempool.Close()
	s.wallet.Close()
	if err := s.blockchain.Close(); err != nil {
		return err
	}
	if err := s.network.Close(); err != nil {
		return err
	}
	if err := s.ds.Close(); err != nil {
		return err
	}
	return nil
}

func (s *Server) makeBlockchainClient(chain *blockchain.Blockchain) *client.InternalClient {
	c := &client.InternalClient{
		BroadcastFunc:                s.submitTransaction,
		GetAccumulatorCheckpointFunc: chain.GetAccumulatorCheckpointByHeight,
		GetBlocksFunc: func(from, to uint32) ([]*blocks.Block, uint32, error) {
			blocks := make([]*blocks.Block, 0, to-from+1)
			for i := from; i <= to; i++ {
				blk, err := chain.GetBlockByHeight(i)
				if err != nil {
					return nil, 0, err
				}
				blocks = append(blocks, blk)
			}
			_, bestHeight, _ := chain.BestBlock()
			return blocks, bestHeight, nil
		},
		SubscribeBlocksFunc: func() (<-chan *blocks.Block, error) {
			ch := make(chan *blocks.Block)
			chain.Subscribe(func(ntf *blockchain.Notification) {
				if blk, ok := ntf.Data.(*blocks.Block); ok && ntf.Type == blockchain.NTBlockConnected {
					ch <- blk
				}
			})
			return ch, nil
		},
		CloseFunc: func() {},
	}
	return c
}

func (s *Server) limitOrphans() {
	if len(s.orphanBlocks) > maxOrphans {
		for id := range s.orphanBlocks {
			delete(s.orphanBlocks, id)
			break
		}
	}
}

func printSplashScreen() {
	colors := []string{
		"\033[35m", // Magenta
		"\033[95m", // Light Magenta
		"\033[94m", // Light Blue
		"\033[34m", // Blue
	}

	// Apply colors to each line
	fmt.Println(colors[0] + ".__.__  .__  .__" + "\033[0m")
	fmt.Println(colors[0] + "|__|  | |  | |__|__ __  _____" + "\033[0m")
	fmt.Println(colors[1] + "|  |  | |  | |  |  |  \\/     \\" + "\033[0m")
	fmt.Println(colors[2] + "|  |  |_|  |_|  |  |  /  Y Y  \\" + "\033[0m")
	fmt.Println(colors[3] + "|__|____/____/__|____/|__|_|  /" + "\033[0m")
	fmt.Println(colors[2] + "                            \\/" + "\033[0m")
}

func (s *Server) printListenAddrs() {
	log.Infof("PeerID: %s", s.network.Host().ID().String())
	var lisAddrs []string
	ifaceAddrs := s.network.Host().Addrs()
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Strings(lisAddrs)
	for _, addr := range lisAddrs {
		log.Infof("Listening on %s", addr)
	}
	log.Infof("gRPC server listening on %s", s.config.RPCOpts.GrpcListener)
}
