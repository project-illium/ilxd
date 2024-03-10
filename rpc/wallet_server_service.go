// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"time"
)

// RegisterViewKey registers a new view key with the server. The server will use this key when
// attempting to decrypt each output. If outputs decrypt, they will be indexed so the client
// can fetch them later.
//
// To free up resources keys will automatically unregister if the wallet has not connected
// in some time.
func (s *GrpcServer) RegisterViewKey(ctx context.Context, req *pb.RegisterViewKeyRequest) (*pb.RegisterViewKeyResponse, error) {
	if s.wsIndex == nil {
		return nil, status.Error(codes.Internal, "wsindex is not active on this server")
	}
	viewKey, err := icrypto.UnmarshalCurve25519PrivateKey(req.ViewKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	exists, err := s.wsIndex.RegistrationExists(s.ds, viewKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Registering again if the registration already exists rolls forward the last seen timestamp
	if err := s.wsIndex.RegisterViewKey(s.ds, viewKey, req.SerializedLockingScript); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Birthday > 0 && !exists {
		checkpoint, height, err := s.chain.GetAccumulatorCheckpointByTimestamp(time.Unix(req.Birthday, 0))
		if err != nil && errors.Is(err, blockchain.ErrNoCheckpoint) {
			return nil, status.Error(codes.Internal, err.Error())
		}

		go s.wsIndex.RescanViewkey(s.ds, viewKey, checkpoint, height, s.chain.GetBlockByHeight)
	}

	return &pb.RegisterViewKeyResponse{}, nil
}

// SubscribeTransactions subscribes to a stream of TransactionsNotifications that match to the
// provided view key.
func (s *GrpcServer) SubscribeTransactions(req *pb.SubscribeTransactionsRequest, stream pb.WalletServerService_SubscribeTransactionsServer) error {
	if s.wsIndex == nil {
		return status.Error(codes.Internal, "wsindex is not active on this server")
	}

	sub := s.wsIndex.Subscribe()
	defer sub.Close()

	keys := make([]crypto.PrivKey, 0, len(req.ViewKeys))
	for _, keyBytes := range req.ViewKeys {
		key, err := crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		keys = append(keys, key)
	}
	for {
		select {
		case userTx := <-sub.C:
			if userTx != nil {
				for _, key := range keys {
					if key.Equals(userTx.ViewKey) {
						err := stream.Send(&pb.TransactionNotification{
							Transaction: userTx.Tx,
							Block_ID:    userTx.BlockID.Bytes(),
							BlockHeight: userTx.BlockHeight,
						})
						if err == io.EOF {
							return nil
						} else if err != nil {
							return status.Error(codes.InvalidArgument, err.Error())
						}
					}
				}
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// GetWalletTransactions returns a list of transactions for the provided view key.
func (s *GrpcServer) GetWalletTransactions(ctx context.Context, req *pb.GetWalletTransactionsRequest) (*pb.GetWalletTransactionsResponse, error) {
	if s.txIndex == nil {
		return nil, status.Error(codes.Internal, "txindex is not active on this server")
	}
	if s.wsIndex == nil {
		return nil, status.Error(codes.Internal, "wsindex is not active on this server")
	}
	viewKey, err := icrypto.UnmarshalCurve25519PrivateKey(req.ViewKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	startHeight := req.GetHeight()
	if req.GetBlock_ID() != nil {
		startHeight, err = s.chain.GetBlockHeight(types.NewID(req.GetBlock_ID()))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	_, tipHeight, _ := s.chain.BestBlock()

	txids, err := s.wsIndex.GetTransactionsIDs(s.ds, viewKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	n := len(txids)
	if req.NbFetch > 0 && int(req.NbFetch) < n {
		n = int(req.NbFetch)
	}
	txs := make([]*transactions.Transaction, 0, n)
	for i := int(req.NbSkip); i < len(txids); i++ {
		blkID, err := s.txIndex.GetContainingBlockID(s.ds, txids[i])
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		height, err := s.chain.GetBlockHeight(blkID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if height < startHeight {
			break
		}
		tx, err := s.txIndex.GetTransaction(s.ds, txids[i])
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		txs = append(txs, tx)
		if len(txs) >= n {
			break
		}
	}

	return &pb.GetWalletTransactionsResponse{
		ChainHeight:  tipHeight,
		Transactions: txs,
	}, nil
}

// GetTxoProof returns the merkle inclusion proof for the given commitment. This information is needed
// by the client to create the zero knowledge proof needed to spend the transaction.
func (s *GrpcServer) GetTxoProof(ctx context.Context, req *pb.GetTxoProofRequest) (*pb.GetTxoProofResponse, error) {
	if s.wsIndex == nil {
		return nil, status.Error(codes.Internal, "wsindex is not active on this server")
	}
	commitments := make([]types.ID, 0, len(req.Commitments))
	for _, commitment := range req.Commitments {
		commitments = append(commitments, types.NewID(commitment))
	}

	proofs, root, err := s.wsIndex.GetTxoProofs(commitments)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	ret := make([]*pb.TxoProof, 0, len(proofs))
	for i, proof := range proofs {
		ret = append(ret, &pb.TxoProof{
			Commitment: commitments[i].Bytes(),
			Hashes:     proof.Hashes,
			Flags:      proof.Flags,
			Index:      proof.Index,
			TxoRoot:    root.Bytes(),
		})
	}
	return &pb.GetTxoProofResponse{
		Proofs: ret,
	}, nil
}
