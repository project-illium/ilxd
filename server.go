// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	badger "github.com/ipfs/go-ds-badger"
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
	"go.uber.org/zap"
	"sort"
	stdsync "sync"
	"time"
)

const maxOrphanDuration = time.Hour
const maxOrphans = 100

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

	orphanBlocks map[types.ID]*orphanBlock
	orphanLock   stdsync.RWMutex

	activeInventory map[types.ID]*blocks.Block
	inventoryLock   stdsync.RWMutex

	inflightRequests map[types.ID]bool
	inflightLock     stdsync.RWMutex
	policy           *policy2.Policy

	ready chan struct{}
}

// BuildServer is the constructor for the server. We pass in the config file here
// and use it to configure all the various parts of the Server.
func BuildServer(config *repo.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := Server{ready: make(chan struct{})}
	defer close(s.ready)

	// Logging
	zapLevel, err := setupLogging(config.LogDir, config.LogLevel, config.Testnet)
	if err != nil {
		return nil, err
	}

	// Policy
	policy := policy2.NewPolicy(
		types.Amount(config.MinFeePerKilobyte),
		types.Amount(config.MinStake),
		config.BlocksizeSoftLimit,
	)

	for _, id := range config.TreasuryWhitelist {
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
	} else if config.Regest {
		netParams = &params.RegestParams
	} else {
		netParams = &params.MainnetParams
	}

	// Setup up badger datastore
	ds, err := badger.NewDatastore(config.DataDir, &badger.DefaultOptions)
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
		if netParams.Name == params.RegestParams.Name {
			privKey, err = crypto.UnmarshalPrivateKey(params.RegtestGenesisKey)
			if err != nil {
				return nil, err
			}
			if err := repo.PutNetworkKey(ds, privKey); err != nil {
				return nil, err
			}
		} else {
			privKey, _, err = repo.GenerateNetworkKeypair()
			if err != nil {
				return nil, err
			}
			if err := repo.PutNetworkKey(ds, privKey); err != nil {
				return nil, err
			}
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

	// Create the blockchain
	sigCache := blockchain.NewSigCache(blockchain.DefaultSigCacheSize)
	proofCache := blockchain.NewProofCache(blockchain.DefaultProofCacheSize)
	var (
		indexerList []indexers.Indexer
		txIndex     *indexers.TxIndex
	)
	if !config.NoTxIndex {
		txIndex = indexers.NewTxIndex()
		indexerList = append(indexerList, txIndex)
	}

	blockchainOpts := []blockchain.Option{
		blockchain.Params(netParams),
		blockchain.Datastore(ds),
		blockchain.MaxNullifiers(blockchain.DefaultMaxNullifiers),
		blockchain.MaxTxoRoots(blockchain.DefaultMaxTxoRoots),
		blockchain.SignatureCache(sigCache),
		blockchain.SnarkProofCache(proofCache),
		blockchain.Indexers(indexerList),
	}

	if config.DropTxIndex {
		if err := indexers.DropTxIndex(ds); err != nil {
			return nil, err
		}
	}

	if !config.NoTxIndex {
		blockchainOpts = append(blockchainOpts, blockchain.Indexers([]indexers.Indexer{indexers.NewTxIndex()}))

	}
	chain, err := blockchain.NewBlockchain(blockchainOpts...)
	if err != nil {
		return nil, err
	}

	// Mempool
	mempoolOpts := []mempool.Option{
		mempool.SignatureCache(sigCache),
		mempool.ProofCache(proofCache),
		mempool.Params(netParams),
		mempool.BlockchainView(chain),
		mempool.MinStake(policy.GetMinStake()),
		mempool.FeePerKilobyte(policy.GetMinFeePerKilobyte()),
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
		net.MempoolValidator(mpool.ProcessTransaction),
		net.MaxBanscore(config.MaxBanscore),
		net.BanDuration(config.BanDuration),
		net.MaxMessageSize(config.MaxMessageSize),
	}
	if config.DisableNATPortMap {
		networkOpts = append(networkOpts, net.DisableNatPortMap())
	}

	network, err := net.NewNetwork(ctx, networkOpts...)
	if err != nil {
		return nil, err
	}

	engine, err := consensus.NewConsensusEngine(ctx, []consensus.Option{
		consensus.Params(netParams),
		consensus.Network(network),
		consensus.Chooser(chain),
		consensus.RequestBlock(s.requestBlock),
		consensus.HasBlock(chain.HasBlock),
	}...)
	if err != nil {
		return nil, err
	}

	grpcServer, err := newGrpcServer(config.RPCOpts, &rpc.GrpcServerConfig{
		Server:           nil,
		HTTPServer:       nil,
		Chain:            chain,
		Network:          network,
		Policy:           policy,
		BroadcastTxFunc:  s.submitTransaction,
		SetLogLevelFunc:  zapLevel.SetLevel,
		ReindexChainFunc: s.reIndexChain,
		RequestBlockFunc: s.requestBlock,
		ChainParams:      netParams,
		Ds:               ds,
		TxMemPool:        mpool,
		TxIndex:          txIndex,
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

	s.ctx = ctx
	s.cancelFunc = cancel
	s.config = config
	s.params = netParams
	s.ds = ds
	s.network = network
	s.blockchain = chain
	s.mempool = mpool
	s.engine = engine
	s.chainService = sync.NewChainService(ctx, s.fetchBlock, chain, network, netParams)
	s.syncManager = sync.NewSyncManager(ctx, chain, network, netParams, s.chainService, s.consensusChoose, s.maybeStartGenerating)
	s.orphanBlocks = make(map[types.ID]*orphanBlock)
	s.activeInventory = make(map[types.ID]*blocks.Block)
	s.inflightRequests = make(map[types.ID]bool)
	s.orphanLock = stdsync.RWMutex{}
	s.inventoryLock = stdsync.RWMutex{}
	s.inflightLock = stdsync.RWMutex{}
	s.policy = policy
	s.generator = generator
	s.grpcServer = grpcServer

	chain.Subscribe(s.handleBlockchainNotification)

	go s.syncManager.Start()

	s.printListenAddrs()

	// If we are the genesis validator then start generating immediately.
	_, height, _ := chain.BestBlock()
	if height == 0 {
		if chain.Validators()[0].PeerID == network.Host().ID() {
			generator.Start()
		}
		s.syncManager.SetCurrent()
	}

	return &s, nil
}

func (s *Server) submitTransaction(tx *transactions.Transaction) error {
	<-s.ready
	err := s.mempool.ProcessTransaction(tx)
	if err != nil {
		return err
	}
	return s.network.BroadcastTransaction(tx)
}

func (s *Server) handleIncomingBlock(xThinnerBlk *blocks.XThinnerBlock, p peer.ID) error {
	<-s.ready
	_, height, _ := s.blockchain.BestBlock()
	if !s.syncManager.IsCurrent() && xThinnerBlk.Header.Height != height+1 {
		return errors.New("not current")
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
		}
	case blockchain.NTAddValidator:
		if pid, ok := ntf.Data.(peer.ID); ok && pid == s.network.Host().ID() {
			if !s.generator.Active() {
				s.generator.Start()
			}
		}
	case blockchain.NTRemoveValidator:
		if pid, ok := ntf.Data.(peer.ID); ok && pid == s.network.Host().ID() {
			if s.generator.Active() {
				s.generator.Close()
			}
		}
	}
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
			s.network.IncreaseBanscore(relayingPeer, 0, 10)
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

	initialPreference, err := s.policy.IsPreferredBlock(blk)
	if err != nil {
		log.Warnf("Error calculating policy preference: %s", err)
	}
	s.inventoryLock.Lock()
	s.activeInventory[blk.ID()] = blk
	s.inventoryLock.Unlock()

	s.orphanLock.Lock()
	delete(s.orphanBlocks, blk.ID())
	s.orphanLock.Unlock()

	s.engine.NewBlock(blk.Header, initialPreference, callback)

	go func(b *blocks.Block, t time.Time) {
		select {
		case status := <-callback:
			switch status {
			case consensus.StatusFinalized:
				blockID := blk.ID()
				log.Debugf("Block %s finalized in %s milliseconds", blockID, time.Since(t).Milliseconds())
				if err := s.blockchain.ConnectBlock(b, blockchain.BFNone); err != nil {
					log.Warnf("Connect block error: block %s: %s", blockID, err)
				} else {
					log.Infof("New block: %s, (height: %d, transactions: %d)", blockID, blk.Header.Height, len(b.Transactions))
					s.syncManager.SetCurrent()
				}
			case consensus.StatusRejected:
				log.Debugf("Block %s rejected by consensus", b.ID())
			}

			s.inventoryLock.Lock()
			delete(s.activeInventory, blk.ID())
			s.inventoryLock.Unlock()

			s.orphanLock.Lock()
			for _, orphan := range s.orphanBlocks {
				if orphan.blk.Header.Height == blk.Header.Height {
					delete(s.orphanBlocks, orphan.blk.ID())
				} else if orphan.blk.Header.Height == blk.Header.Height+1 {
					s.processBlock(orphan.blk, orphan.relayingPeer, false)
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
	if len(missing) == 0 {
		return nil, errors.New("block invalid")
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
	s.inflightLock.RLock()

	s.inflightLock.Lock()
	s.inflightRequests[blockID] = true
	s.inflightLock.Unlock()

	blk, err := s.chainService.GetBlock(remotePeer, blockID)
	if err != nil {
		s.network.IncreaseBanscore(remotePeer, 34, 0)
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

func (s *Server) maybeStartGenerating() {
	<-s.ready
	_, err := s.blockchain.GetValidator(s.network.Host().ID())
	if err == nil {
		s.generator.Start()
	}
}

// Close shuts down all the parts of the server and blocks until
// they finish closing.
func (s *Server) Close() error {
	<-s.ready
	s.cancelFunc()
	if err := s.network.Close(); err != nil {
		return err
	}
	if err := s.ds.Close(); err != nil {
		return err
	}
	if err := s.blockchain.Close(); err != nil {
		return err
	}
	s.generator.Close()
	s.syncManager.Close()
	s.engine.Close()
	s.mempool.Close()
	return nil
}

func (s *Server) limitOrphans() {
	if len(s.orphanBlocks) > maxOrphans {
		for id := range s.orphanBlocks {
			delete(s.orphanBlocks, id)
			break
		}
	}
}

func (s *Server) printListenAddrs() {
	fmt.Println(`.__.__  .__  .__`)
	fmt.Println(`|__|  | |  | |__|__ __  _____`)
	fmt.Println(`|  |  | |  | |  |  |  \/     \`)
	fmt.Println(`|  |  |_|  |_|  |  |  /  Y Y  \`)
	fmt.Println(`|__|____/____/__|____/|__|_|  /`)
	fmt.Println(`                            \/`)

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
