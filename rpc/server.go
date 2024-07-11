// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/indexers"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/walletlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net/http"
	"sync"
)

var _ pb.BlockchainServiceServer = (*GrpcServer)(nil)
var _ pb.NodeServiceServer = (*GrpcServer)(nil)
var _ pb.WalletServiceServer = (*GrpcServer)(nil)
var _ pb.WalletServerServiceServer = (*GrpcServer)(nil)
var _ pb.ProverServiceServer = (*GrpcServer)(nil)

// GrpcServerConfig hols the various objects needed by the GrpcServer to
// perform its functions.
type GrpcServerConfig struct {
	Server     *grpc.Server
	HTTPServer *http.Server

	Chain                *blockchain.Blockchain
	Network              *net.Network
	Wallet               *walletlib.Wallet
	Prover               zk.Prover
	Policy               *policy.Policy
	BroadcastTxFunc      func(tx *transactions.Transaction) error
	SetLogLevelFunc      func(level string) error
	ReindexChainFunc     func() error
	RequestBlockFunc     func(blockID types.ID, remotePeer peer.ID)
	AutoStakeFunc        func(bool) error
	NetworkKeyFunc       func() (crypto.PrivKey, error)
	ChainParams          *params.NetworkParams
	Ds                   datastore.Datastore
	TxMemPool            *mempool.Mempool
	DisableNodeService   bool
	DisableWalletService bool
	DisableWalletServer  bool
	DisableProverServer  bool

	TxIndex   *indexers.TxIndex
	WSIndex   *indexers.WalletServerIndex
	AddrIndex *indexers.AddrIndex
}

// GrpcServer is the gRPC server implementation. It holds all the objects
// necessary to serve the RPCs and implements the ilxdrpc.proto interface.
type GrpcServer struct {
	chain            *blockchain.Blockchain
	chainParams      *params.NetworkParams
	ds               datastore.Datastore
	txMemPool        *mempool.Mempool
	network          *net.Network
	policy           *policy.Policy
	wallet           *walletlib.Wallet
	prover           zk.Prover
	broadcastTxFunc  func(tx *transactions.Transaction) error
	setLogLevelFunc  func(level string) error
	reindexChainFunc func() error
	requestBlockFunc func(blockID types.ID, remotePeer peer.ID)
	autoStakeFunc    func(bool) error
	networkKeyFunc   func() (crypto.PrivKey, error)

	txIndex              *indexers.TxIndex
	wsIndex              *indexers.WalletServerIndex
	addrIndex            *indexers.AddrIndex
	provingServiceActive bool

	httpServer *http.Server
	subs       map[types.ID]*subscription
	subMtx     sync.RWMutex
	events     chan interface{}
	quit       chan struct{}

	pb.UnimplementedBlockchainServiceServer
	pb.UnimplementedNodeServiceServer
	pb.UnimplementedWalletServiceServer
	pb.UnimplementedWalletServerServiceServer
	pb.UnimplementedProverServiceServer
}

// NewGrpcServer returns a new GrpcServer which has not yet
// be started.
func NewGrpcServer(cfg *GrpcServerConfig) *GrpcServer {
	s := &GrpcServer{
		chain:                cfg.Chain,
		chainParams:          cfg.ChainParams,
		ds:                   cfg.Ds,
		txMemPool:            cfg.TxMemPool,
		network:              cfg.Network,
		wallet:               cfg.Wallet,
		prover:               cfg.Prover,
		broadcastTxFunc:      cfg.BroadcastTxFunc,
		setLogLevelFunc:      cfg.SetLogLevelFunc,
		reindexChainFunc:     cfg.ReindexChainFunc,
		requestBlockFunc:     cfg.RequestBlockFunc,
		autoStakeFunc:        cfg.AutoStakeFunc,
		networkKeyFunc:       cfg.NetworkKeyFunc,
		txIndex:              cfg.TxIndex,
		wsIndex:              cfg.WSIndex,
		addrIndex:            cfg.AddrIndex,
		provingServiceActive: !cfg.DisableProverServer,
		policy:               cfg.Policy,
		httpServer:           cfg.HTTPServer,
		subs:                 make(map[types.ID]*subscription),
		subMtx:               sync.RWMutex{},
		events:               make(chan interface{}),
		quit:                 make(chan struct{}),
	}
	reflection.Register(cfg.Server)
	pb.RegisterBlockchainServiceServer(cfg.Server, s)
	if !cfg.DisableNodeService {
		pb.RegisterNodeServiceServer(cfg.Server, s)
	}
	if !cfg.DisableWalletService {
		pb.RegisterWalletServiceServer(cfg.Server, s)
	}
	if !cfg.DisableWalletServer {
		pb.RegisterWalletServerServiceServer(cfg.Server, s)
	}
	if !cfg.DisableProverServer {
		pb.RegisterProverServiceServer(cfg.Server, s)
	}

	s.chain.Subscribe(s.handleBlockchainNotifications)

	return s
}

func (s *GrpcServer) Close() {
	close(s.quit)
}

type subscription struct {
	C    chan interface{}
	quit chan struct{}
}

func (sub *subscription) Close() {
	close(sub.quit)
}

func (s *GrpcServer) handleBlockchainNotifications(n *blockchain.Notification) {
	s.subMtx.RLock()
	defer s.subMtx.RUnlock()

	for _, sub := range s.subs {
		sub.C <- n
	}
}

func (s *GrpcServer) subscribeEvents() *subscription {
	sub := &subscription{
		C:    make(chan interface{}),
		quit: make(chan struct{}),
	}
	b := make([]byte, 32)
	rand.Read(b)
	s.subMtx.Lock()
	s.subs[types.NewID(b)] = sub
	s.subMtx.Unlock()

	go func(sb *subscription, id types.ID) {
		<-sb.quit
		s.subMtx.Lock()
		go func() {
			for range sub.C {
			}
		}()
		close(sub.C)
		delete(s.subs, id)
		s.subMtx.Unlock()
	}(sub, types.NewID(b))
	return sub
}
