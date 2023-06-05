// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

const maxBatchSize = 2000

// GetMempoolInfo returns the state of the current mempool
func (s *GrpcServer) GetMempoolInfo(ctx context.Context, req *pb.GetMempoolInfoRequest) (*pb.GetMempoolInfoResponse, error) {
	size := 0
	bytes := 0
	for _, tx := range s.txMemPool.GetTransactions() {
		size++
		n, err := tx.SerializedSize()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		bytes += n
	}
	return &pb.GetMempoolInfoResponse{
		Size:  uint32(size),
		Bytes: uint32(bytes),
	}, nil
}

// GetMempool returns all the transactions in the mempool
func (s *GrpcServer) GetMempool(ctx context.Context, req *pb.GetMempoolRequest) (*pb.GetMempoolResponse, error) {

	txs := s.txMemPool.GetTransactions()
	td := make([]*pb.TransactionData, 0, len(txs))
	for _, tx := range txs {
		if req.FullTransactions {
			td = append(td, &pb.TransactionData{
				TxidsOrTxs: &pb.TransactionData_Transaction{
					Transaction: tx,
				},
			})
		} else {
			id := tx.ID()
			td = append(td, &pb.TransactionData{
				TxidsOrTxs: &pb.TransactionData_Transaction_ID{
					Transaction_ID: id[:],
				},
			})
		}
	}

	return &pb.GetMempoolResponse{
		TransactionData: td,
	}, nil
}

// GetBlockchainInfo returns data about the blockchain including the most recent block hash and height.
func (s *GrpcServer) GetBlockchainInfo(ctx context.Context, req *pb.GetBlockchainInfoRequest) (*pb.GetBlockchainInfoResponse, error) {
	var nt pb.GetBlockchainInfoResponse_Network
	switch s.chainParams.Name {
	case params.MainnetParams.Name:
		nt = pb.GetBlockchainInfoResponse_MAINNET
	case params.Testnet1Params.Name:
		nt = pb.GetBlockchainInfoResponse_TESTNET
	case params.RegestParams.Name:
		nt = pb.GetBlockchainInfoResponse_REGTEST
	default:
		return nil, status.Error(codes.Internal, "unknown network params")
	}

	id, height, ts := s.chain.BestBlock()

	currentSupply, err := s.chain.CurrentSupply()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	totalStaked := s.chain.TotalStaked()

	return &pb.GetBlockchainInfoResponse{
		Network:          nt,
		BestHeight:       height,
		BestBlock_ID:     id[:],
		BlockTime:        ts.Unix(),
		TxIndex:          s.txIndex != nil,
		CiculatingSupply: uint64(currentSupply),
		TotalStaked:      uint64(totalStaked),
	}, nil
}

// GetBlockInfo returns a BlockHeader plus some extra metadata.
func (s *GrpcServer) GetBlockInfo(ctx context.Context, req *pb.GetBlockInfoRequest) (*pb.GetBlockInfoResponse, error) {
	var (
		blk *blocks.Block
		err error
	)
	if len(req.GetBlock_ID()) == 0 {
		blk, err = s.chain.GetBlockByHeight(req.GetHeight())
	} else {
		blk, err = s.chain.GetBlockByID(types.NewID(req.GetBlock_ID()))
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	id := blk.ID()
	size, err := blk.SerializedSize()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &pb.GetBlockInfoResponse{
		Info: &pb.BlockInfo{
			Block_ID:    id[:],
			Version:     blk.Header.Version,
			Height:      blk.Header.Height,
			Parent:      blk.Header.Parent,
			Child:       nil,
			Timestamp:   blk.Header.Timestamp,
			TxRoot:      blk.Header.TxRoot,
			Producer_ID: blk.Header.Producer_ID,
			Size:        uint32(size),
			NumTxs:      uint32(len(blk.Transactions)),
		},
	}
	child, err := s.chain.GetHeaderByHeight(blk.Header.Height + 1)
	if err == nil {
		childID := child.ID()
		resp.Info.Child = childID[:]
	}

	return resp, nil
}

// GetBlock returns the detailed data for a block.
func (s *GrpcServer) GetBlock(ctx context.Context, req *pb.GetBlockRequest) (*pb.GetBlockResponse, error) {
	var (
		blk *blocks.Block
		err error
	)
	if len(req.GetBlock_ID()) == 0 {
		blk, err = s.chain.GetBlockByHeight(req.GetHeight())
	} else {
		blk, err = s.chain.GetBlockByID(types.NewID(req.GetBlock_ID()))
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.GetBlockResponse{
		Block: blk,
	}, nil
}

// GetCompressedBlock returns a block that is stripped down to just the outputs.
func (s *GrpcServer) GetCompressedBlock(ctx context.Context, req *pb.GetCompressedBlockRequest) (*pb.GetCompressedBlockResponse, error) {
	var (
		blk *blocks.Block
		err error
	)
	if len(req.GetBlock_ID()) == 0 {
		blk, err = s.chain.GetBlockByHeight(req.GetHeight())
	} else {
		blk, err = s.chain.GetBlockByID(types.NewID(req.GetBlock_ID()))
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &pb.GetCompressedBlockResponse{
		Block: &blocks.CompressedBlock{
			Height:  blk.Header.Height,
			Outputs: blk.Outputs(),
		},
	}, nil
}

// GetHeaders returns a batch of headers according to the request parameters.
func (s *GrpcServer) GetHeaders(ctx context.Context, req *pb.GetHeadersRequest) (*pb.GetHeadersResponse, error) {
	endHeight := req.EndHeight
	if endHeight-req.StartHeight+1 > 2000 {
		endHeight = req.StartHeight + 1999
	}
	_, bestHeight, _ := s.chain.BestBlock()
	if endHeight > bestHeight {
		endHeight = bestHeight
	}
	headers := make([]*blocks.BlockHeader, 0, endHeight-req.StartHeight+1)
	for i := req.StartHeight; i <= endHeight; i++ {
		header, err := s.chain.GetHeaderByHeight(i)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		headers = append(headers, header)
	}
	return &pb.GetHeadersResponse{
		Headers: headers,
	}, nil
}

// GetCompressedBlocks returns a batch of CompressedBlocks according to the request parameters.
func (s *GrpcServer) GetCompressedBlocks(ctx context.Context, req *pb.GetCompressedBlocksRequest) (*pb.GetCompressedBlocksResponse, error) {
	endHeight := req.EndHeight
	if endHeight-req.StartHeight+1 > 2000 {
		endHeight = req.StartHeight + 1999
	}
	_, bestHeight, _ := s.chain.BestBlock()
	if endHeight > bestHeight {
		endHeight = bestHeight
	}
	blks := make([]*blocks.CompressedBlock, 0, endHeight-req.StartHeight+1)
	for i := req.StartHeight; i <= endHeight; i++ {
		blk, err := s.chain.GetBlockByHeight(i)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		blks = append(blks, &blocks.CompressedBlock{
			Height:  blk.Header.Height,
			Outputs: blk.Outputs(),
		})
	}
	return &pb.GetCompressedBlocksResponse{
		Blocks: blks,
	}, nil
}

// GetTransaction returns the transaction for the given transaction ID.
func (s *GrpcServer) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	if s.txIndex == nil {
		return nil, status.Error(codes.Unavailable, "tx index is not available")
	}

	tx, err := s.txIndex.GetTransaction(s.ds, types.NewID(req.Transaction_ID))
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.GetTransactionResponse{
		Tx: tx,
	}, nil
}

// GetMerkleProof returns a Merkle (SPV) proof for a specific transaction in the provided block.
func (s *GrpcServer) GetMerkleProof(ctx context.Context, req *pb.GetMerkleProofRequest) (*pb.GetMerkleProofResponse, error) {
	if s.txIndex == nil {
		return nil, status.Error(codes.Unavailable, "tx index is not available")
	}
	blockID, err := s.txIndex.GetContainingBlockID(s.ds, types.NewID(req.Transaction_ID))
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	blk, err := s.chain.GetBlockByID(blockID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	id := blk.ID()
	size, err := blk.SerializedSize()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	merkles := blockchain.BuildMerkleTreeStore(blk.Transactions)
	hashes, flags := blockchain.MerkleInclusionProof(merkles, types.NewID(req.Transaction_ID))
	resp := &pb.GetMerkleProofResponse{
		Block: &pb.BlockInfo{
			Block_ID:    id[:],
			Version:     blk.Header.Version,
			Height:      blk.Header.Height,
			Parent:      blk.Header.Parent,
			Child:       nil,
			Timestamp:   blk.Header.Timestamp,
			TxRoot:      blk.Header.TxRoot,
			Producer_ID: blk.Header.Producer_ID,
			Size:        uint32(size),
			NumTxs:      uint32(len(blk.Transactions)),
		},
		Hashes: hashes,
		Flags:  flags,
	}
	child, err := s.chain.GetHeaderByHeight(blk.Header.Height + 1)
	if err == nil {
		childID := child.ID()
		resp.Block.Child = childID[:]
	}
	return resp, nil
}

// GetValidator returns all the information about the given validator including number of staked coins.
func (s *GrpcServer) GetValidator(ctx context.Context, req *pb.GetValidatorRequest) (*pb.GetValidatorResponse, error) {
	pid, err := peer.IDFromBytes(req.Validator_ID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	validator, err := s.chain.GetValidator(pid)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	resp := &pb.GetValidatorResponse{
		Validator: &pb.Validator{
			Validator_ID:   req.Validator_ID,
			TotalStake:     uint64(validator.TotalStake),
			UnclaimedCoins: uint64(validator.UnclaimedCoins),
			EpochBlocks:    validator.EpochBlocks,
		},
	}
	for nullifier := range validator.Nullifiers {
		resp.Validator.Nullifiers = append(resp.Validator.Nullifiers, nullifier[:])
	}
	return resp, nil
}

// GetValidatorSetInfo returns information about the validator set.
func (s *GrpcServer) GetValidatorSetInfo(ctx context.Context, req *pb.GetValidatorSetInfoRequest) (*pb.GetValidatorSetInfoResponse, error) {
	return &pb.GetValidatorSetInfoResponse{
		TotalStaked:   uint64(s.chain.TotalStaked()),
		NumValidators: uint32(s.chain.ValidatorSetSize()),
	}, nil
}

// GetValidatorSet returns all the validators in the current validator set.
func (s *GrpcServer) GetValidatorSet(ctx context.Context, req *pb.GetValidatorSetRequest) (*pb.GetValidatorSetResponse, error) {
	validators := s.chain.Validators()
	resp := &pb.GetValidatorSetResponse{
		Validators: make([]*pb.Validator, 0, len(validators)),
	}
	for _, v := range validators {
		valID, err := v.PeerID.Marshal()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		val := &pb.Validator{
			Validator_ID:   valID,
			TotalStake:     uint64(v.TotalStake),
			UnclaimedCoins: uint64(v.UnclaimedCoins),
			EpochBlocks:    v.EpochBlocks,
		}
		for nullifier := range v.Nullifiers {
			val.Nullifiers = append(val.Nullifiers, nullifier[:])
		}
		resp.Validators = append(resp.Validators, val)
	}
	return resp, nil
}

// GetAccumulatorCheckpoint returns the accumulator at the requested height.
func (s *GrpcServer) GetAccumulatorCheckpoint(ctx context.Context, req *pb.GetAccumulatorCheckpointRequest) (*pb.GetAccumulatorCheckpointResponse, error) {
	var (
		accumulator *blockchain.Accumulator
		height      uint32
		err         error
	)
	if req.GetTimestamp() == 0 {
		accumulator, height, err = s.chain.GetAccumulatorCheckpointByHeight(req.GetHeight())
	} else {
		accumulator, height, err = s.chain.GetAccumulatorCheckpointByTimestamp(time.Unix(req.GetTimestamp(), 0))
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.GetAccumulatorCheckpointResponse{
		Height:      height,
		NumEntries:  accumulator.NumElements(),
		Accumulator: accumulator.Hashes(),
	}, nil
}

// SubmitTransaction validates a transaction and submits it to the network. An error will be returned if it fails validation.
func (s *GrpcServer) SubmitTransaction(ctx context.Context, req *pb.SubmitTransactionRequest) (*pb.SubmitTransactionResponse, error) {
	err := s.broadcastTxFunc(req.Transaction)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	txid := req.Transaction.ID()
	return &pb.SubmitTransactionResponse{
		Transaction_ID: txid[:],
	}, nil
}

// SubscribeBlocks returns a stream of notifications when new blocks are finalized and connected to the chain.
func (s *GrpcServer) SubscribeBlocks(req *pb.SubscribeBlocksRequest, stream pb.BlockchainService_SubscribeBlocksServer) error {
	sub := s.subscribeEvents()

	for {
		select {
		case <-s.quit:
			return nil
		case n := <-sub.C:
			if notif, ok := n.(*blockchain.Notification); ok {
				if notif.Type == blockchain.NTBlockConnected {
					blk, ok := notif.Data.(*blocks.Block)
					if !ok {
						continue
					}
					id := blk.Header.ID()
					size, err := blk.SerializedSize()
					if err != nil {
						continue
					}
					resp := &pb.BlockNotification{
						BlockInfo: &pb.BlockInfo{
							Block_ID:    id[:],
							Version:     blk.Header.Version,
							Height:      blk.Header.Height,
							Parent:      blk.Header.Parent,
							Child:       nil,
							Timestamp:   blk.Header.Timestamp,
							TxRoot:      blk.Header.TxRoot,
							Producer_ID: blk.Header.Producer_ID,
							Size:        uint32(size),
							NumTxs:      uint32(len(blk.Transactions)),
						},
						Transactions: make([]*pb.TransactionData, 0, len(blk.Transactions)),
					}
					if req.FullBlock {
						for _, tx := range blk.Transactions {
							if req.FullTransactions {
								resp.Transactions = append(resp.Transactions, &pb.TransactionData{
									TxidsOrTxs: &pb.TransactionData_Transaction{
										Transaction: tx,
									},
								})
							} else {
								id := tx.ID()
								resp.Transactions = append(resp.Transactions, &pb.TransactionData{
									TxidsOrTxs: &pb.TransactionData_Transaction_ID{
										Transaction_ID: id[:],
									},
								})
							}
						}
					}
					if err := stream.Send(resp); err != nil {
						return err
					}
				}
			}
		}
	}
}
