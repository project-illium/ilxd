// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/rand"
)

// GetHostInfo returns info about the libp2p host
func (s *GrpcServer) GetHostInfo(ctx context.Context, req *pb.GetHostInfoRequest) (*pb.GetHostInfoResponse, error) {
	maaddrs := s.network.Host().Addrs()
	addrs := make([]string, 0, len(maaddrs))
	for _, addr := range maaddrs {
		addrs = append(addrs, addr.String())
	}
	return &pb.GetHostInfoResponse{
		Peer_ID:      s.network.Host().ID().String(),
		Addrs:        addrs,
		Peers:        uint32(len(s.network.Host().Network().Peers())),
		TxIndex:      s.txIndex != nil,
		WalletServer: s.wsIndex != nil,
		Reachability: s.network.Reachability().String(),
	}, nil
}

// GetNetworkKey returns the node's network private key
func (s *GrpcServer) GetNetworkKey(ctx context.Context, req *pb.GetNetworkKeyRequest) (*pb.GetNetworkKeyResponse, error) {
	key, err := s.networkKeyFunc()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	b, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetNetworkKeyResponse{
		NetworkPrivateKey: b,
	}, nil
}

// GetPeers returns a list of peers that this node is connected to
func (s *GrpcServer) GetPeers(ctx context.Context, req *pb.GetPeersRequest) (*pb.GetPeersResponse, error) {
	peers := s.network.Host().Network().Peers()
	ret := make([]*pb.Peer, 0, len(peers))
	for _, p := range peers {
		userAgentBytes, err := s.network.Host().Peerstore().Get(p, "AgentVersion")
		if err != nil {
			continue
		}
		userAgentString, ok := (userAgentBytes).(string)
		if !ok {
			continue
		}
		maaddrs := s.network.Host().Peerstore().Addrs(p)
		addrs := make([]string, 0, len(maaddrs))
		for _, addr := range maaddrs {
			addrs = append(addrs, addr.String())
		}
		ret = append(ret, &pb.Peer{
			Id:        p.String(),
			UserAgent: userAgentString,
			Addrs:     addrs,
		})
	}
	return &pb.GetPeersResponse{
		Peers: ret,
	}, nil
}

// AddPeer attempts to connect to the provided peer
func (s *GrpcServer) AddPeer(ctx context.Context, req *pb.AddPeerRequest) (*pb.AddPeerResponse, error) {
	peerID, err := peer.Decode(req.Peer_ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = s.network.Host().Connect(ctx, peer.AddrInfo{ID: peerID})
	if err != nil {
		return nil, err
	}
	return &pb.AddPeerResponse{}, nil
}

// BlockPeer blocks the given peer for the provided time period
func (s *GrpcServer) BlockPeer(ctx context.Context, req *pb.BlockPeerRequest) (*pb.BlockPeerResponse, error) {
	peerID, err := peer.Decode(req.Peer_ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = s.network.ConnGater().BlockPeer(peerID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.BlockPeerResponse{}, nil
}

// UnblockPeer removes a peer from the block list
func (s *GrpcServer) UnblockPeer(ctx context.Context, req *pb.UnblockPeerRequest) (*pb.UnblockPeerResponse, error) {
	peerID, err := peer.Decode(req.Peer_ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = s.network.ConnGater().UnblockPeer(peerID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.UnblockPeerResponse{}, nil
}

// SetLogLevel changes the logging level of the node
func (s *GrpcServer) SetLogLevel(ctx context.Context, req *pb.SetLogLevelRequest) (*pb.SetLogLevelResponse, error) {
	var logLevelSeverity = map[pb.SetLogLevelRequest_Level]string{
		pb.SetLogLevelRequest_TRACE:   "TRACE",
		pb.SetLogLevelRequest_DEBUG:   "DEBUG",
		pb.SetLogLevelRequest_INFO:    "INFO",
		pb.SetLogLevelRequest_WARNING: "WARNING",
		pb.SetLogLevelRequest_ERROR:   "ERROR",
		pb.SetLogLevelRequest_FATAL:   "FATAL",
	}
	level, ok := logLevelSeverity[req.Level]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unknown log level")
	}
	if err := s.setLogLevelFunc(level); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.SetLogLevelResponse{}, nil
}

// GetMinFeePerKilobyte returns the node's current minimum transaction fee needed to relay transactions and admit them into the mempool.
func (s *GrpcServer) GetMinFeePerKilobyte(ctx context.Context, req *pb.GetMinFeePerKilobyteRequest) (*pb.GetMinFeePerKilobyteResponse, error) {
	return &pb.GetMinFeePerKilobyteResponse{
		FeePerKilobyte: uint64(s.policy.GetMinFeePerKilobyte()),
	}, nil
}

// SetMinFeePerKilobyte sets the node's fee policy
func (s *GrpcServer) SetMinFeePerKilobyte(ctx context.Context, req *pb.SetMinFeePerKilobyteRequest) (*pb.SetMinFeePerKilobyteResponse, error) {
	s.policy.SetMinFeePerKilobyte(types.Amount(req.FeePerKilobyte))
	return &pb.SetMinFeePerKilobyteResponse{}, nil
}

// GetMinStake returns the node's current minimum stake policy.
func (s *GrpcServer) GetMinStake(ctx context.Context, req *pb.GetMinStakeRequest) (*pb.GetMinStakeResponse, error) {
	return &pb.GetMinStakeResponse{
		MinStakeAmount: uint64(s.policy.GetMinStake()),
	}, nil
}

// SetMinStake sets the node's minimum stake policy
func (s *GrpcServer) SetMinStake(ctx context.Context, req *pb.SetMinStakeRequest) (*pb.SetMinStakeResponse, error) {
	s.policy.SetMinStake(types.Amount(req.MinStakeAmount))
	return &pb.SetMinStakeResponse{}, nil
}

// GetBlockSizeSoftLimit returns the node's current blocksize soft limit.
func (s *GrpcServer) GetBlockSizeSoftLimit(ctx context.Context, req *pb.GetBlockSizeSoftLimitRequest) (*pb.GetBlockSizeSoftLimitResponse, error) {
	return &pb.GetBlockSizeSoftLimitResponse{
		BlockSize: s.policy.GetBlocksizeSoftLimit(),
	}, nil
}

// SetBlockSizeSoftLimit sets the node's blocksize soft limit policy.
func (s *GrpcServer) SetBlockSizeSoftLimit(ctx context.Context, req *pb.SetBlockSizeSoftLimitRequest) (*pb.SetBlockSizeSoftLimitResponse, error) {
	s.policy.SetBlocksizeSoftLimit(req.BlockSize)
	return &pb.SetBlockSizeSoftLimitResponse{}, nil
}

// GetTreasuryWhitelist returns the current treasury whitelist for the node.
func (s *GrpcServer) GetTreasuryWhitelist(ctx context.Context, req *pb.GetTreasuryWhitelistRequest) (*pb.GetTreasuryWhitelistResponse, error) {
	whitelist := s.policy.GetTreasuryWhitelist()
	ret := make([][]byte, 0, len(whitelist))
	for _, txid := range whitelist {
		ret = append(ret, txid[:])
	}
	return &pb.GetTreasuryWhitelistResponse{
		Txids: ret,
	}, nil
}

// UpdateTreasuryWhitelist adds or removes a transaction to from the treasury whitelist
func (s *GrpcServer) UpdateTreasuryWhitelist(ctx context.Context, req *pb.UpdateTreasuryWhitelistRequest) (*pb.UpdateTreasuryWhitelistResponse, error) {
	if len(req.Add) > 0 {
		toAdd := make([]types.ID, 0, len(req.Add))
		for _, txid := range req.Add {
			toAdd = append(toAdd, types.NewID(txid))
		}
		s.policy.AddToTreasuryWhitelist(toAdd...)
	}
	if len(req.Add) > 0 {
		toRemove := make([]types.ID, 0, len(req.Remove))
		for _, txid := range req.Remove {
			toRemove = append(toRemove, types.NewID(txid))
		}
		s.policy.RemoveFromTreasuryWhitelist(toRemove...)
	}
	return &pb.UpdateTreasuryWhitelistResponse{}, nil
}

// ReconsiderBlock tries to reprocess the given block
func (s *GrpcServer) ReconsiderBlock(ctx context.Context, req *pb.ReconsiderBlockRequest) (*pb.ReconsiderBlockResponse, error) {
	var (
		p   peer.ID
		err error
	)
	if req.DownloadPeer != "" {
		p, err = peer.Decode(req.DownloadPeer)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	} else {
		peers := s.network.Host().Network().Peers()
		p = peers[rand.Intn(len(peers))]
	}
	s.requestBlockFunc(types.NewID(req.Block_ID), p)
	return &pb.ReconsiderBlockResponse{}, nil
}

// RecomputeChainState deletes the accumulator, validator set, and nullifier set and rebuilds them by
// loading and re-processing all blocks from genesis.
func (s *GrpcServer) RecomputeChainState(ctx context.Context, req *pb.RecomputeChainStateRequest) (*pb.RecomputeChainStateResponse, error) {
	go s.reindexChainFunc() //nolint:errcheck
	return &pb.RecomputeChainStateResponse{}, nil
}
