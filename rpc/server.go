// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/indexers"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/rpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net/http"
	"sync"
)

// GrpcServerConfig hols the various objects needed by the GrpcServer to
// perform its functions.
type GrpcServerConfig struct {
	Server     *grpc.Server
	HTTPServer *http.Server

	Chain       *blockchain.Blockchain
	ChainParams *params.NetworkParams
	Ds          repo.Datastore
	TxMemPool   *mempool.Mempool

	TxIndex *indexers.TxIndex
}

// GrpcServer is the gRPC server implementation. It holds all the objects
// necessary to serve the RPCs and implements the bchrpc.proto interface.
type GrpcServer struct {
	chain       *blockchain.Blockchain
	chainParams *params.NetworkParams
	ds          repo.Datastore
	txMemPool   *mempool.Mempool
	network     *net.Network

	txIndex *indexers.TxIndex

	httpServer *http.Server
	events     chan interface{}
	quit       chan struct{}

	wg       sync.WaitGroup
	ready    uint32 // atomic
	shutdown int32  // atomic
}

// NewGrpcServer returns a new GrpcServer which has not yet
// be started.
func NewGrpcServer(cfg *GrpcServerConfig) *GrpcServer {
	s := &GrpcServer{
		chain:       cfg.Chain,
		chainParams: cfg.ChainParams,
		ds:          cfg.Ds,
		txMemPool:   cfg.TxMemPool,
		txIndex:     cfg.TxIndex,
		httpServer:  cfg.HTTPServer,
		events:      make(chan interface{}),
		quit:        make(chan struct{}),
		wg:          sync.WaitGroup{},
	}
	reflection.Register(cfg.Server)
	pb.RegisterBlockchainServiceServer(cfg.Server, s)
	return s
}
