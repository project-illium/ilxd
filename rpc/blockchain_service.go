// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"errors"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/walletlib"
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
	case params.AlphanetParams.Name:
		nt = pb.GetBlockchainInfoResponse_ALPHANET
	default:
		return nil, status.Error(codes.Internal, "unknown network params")
	}

	id, height, ts := s.chain.BestBlock()

	currentSupply, err := s.chain.CurrentSupply()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	totalStaked := s.chain.TotalStaked()

	treasuryBal, err := s.chain.TreasuryBalance()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetBlockchainInfoResponse{
		Network:           nt,
		BestHeight:        height,
		BestBlock_ID:      id[:],
		BlockTime:         ts.Unix(),
		TxIndex:           s.txIndex != nil,
		CirculatingSupply: uint64(currentSupply),
		TotalStaked:       uint64(totalStaked),
		TreasuryBalance:   uint64(treasuryBal),
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

	cb := &blocks.CompressedBlock{
		Height: blk.Header.Height,
		Txs:    make([]*blocks.CompressedBlock_CompressedTx, 0, len(blk.Transactions)),
	}
	for _, tx := range blk.Transactions {
		nullifiers := make([][]byte, 0, len(tx.Nullifiers()))
		for _, n := range tx.Nullifiers() {
			nullifiers = append(nullifiers, n.Bytes())
		}
		cb.Txs = append(cb.Txs, &blocks.CompressedBlock_CompressedTx{
			Txid:       tx.ID().Bytes(),
			Nullifiers: nullifiers,
			Outputs:    tx.Outputs(),
		})
	}

	return &pb.GetCompressedBlockResponse{
		Block: cb,
	}, nil
}

// GetHeaders returns a batch of headers according to the request parameters.
func (s *GrpcServer) GetHeaders(ctx context.Context, req *pb.GetHeadersRequest) (*pb.GetHeadersResponse, error) {
	endHeight := req.EndHeight
	if endHeight-req.StartHeight+1 > maxBatchSize {
		endHeight = req.StartHeight + maxBatchSize - 1
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
	if endHeight-req.StartHeight+1 > maxBatchSize || endHeight <= 0 {
		endHeight = req.StartHeight + maxBatchSize - 1
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
		cb := &blocks.CompressedBlock{
			Height: blk.Header.Height,
			Txs:    make([]*blocks.CompressedBlock_CompressedTx, 0, len(blk.Transactions)),
		}
		for _, tx := range blk.Transactions {
			nullifiers := make([][]byte, 0, len(tx.Nullifiers()))
			for _, n := range tx.Nullifiers() {
				nullifiers = append(nullifiers, n.Bytes())
			}
			cb.Txs = append(cb.Txs, &blocks.CompressedBlock_CompressedTx{
				Txid:       tx.ID().Bytes(),
				Nullifiers: nullifiers,
				Outputs:    tx.Outputs(),
			})
		}
		blks = append(blks, cb)
	}
	return &pb.GetCompressedBlocksResponse{
		Blocks: blks,
	}, nil
}

// GetTransaction returns the transaction for the given transaction ID.
//
// **Requires TxIndex**
// **Input/Output metadata requires AddrIndex**
func (s *GrpcServer) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	if s.txIndex == nil {
		return nil, status.Error(codes.Unavailable, "tx index is not available")
	}

	tx, blockID, err := s.txIndex.GetTransaction(s.ds, types.NewID(req.Transaction_ID))
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	height, err := s.chain.GetBlockHeight(blockID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &pb.GetTransactionResponse{
		Tx:       tx,
		Block_ID: blockID.Bytes(),
		Height:   height,
	}
	if s.addrIndex != nil {
		metadata, err := s.addrIndex.GetTransactionMetadata(s.ds, tx.ID())
		if err != nil && !errors.Is(err, datastore.ErrNotFound) {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if errors.Is(err, datastore.ErrNotFound) {
			resp.Inputs = make([]*pb.IOMetadata, len(tx.Nullifiers()))
			for i := 0; i < len(resp.Inputs); i++ {
				resp.Inputs[i] = &pb.IOMetadata{
					IoType: &pb.IOMetadata_Unknown_{
						Unknown: &pb.IOMetadata_Unknown{},
					},
				}
			}
			resp.Outputs = make([]*pb.IOMetadata, len(tx.Outputs()))
			for i := 0; i < len(resp.Outputs); i++ {
				resp.Outputs[i] = &pb.IOMetadata{
					IoType: &pb.IOMetadata_Unknown_{
						Unknown: &pb.IOMetadata_Unknown{},
					},
				}
			}
		} else {
			resp.Inputs = make([]*pb.IOMetadata, len(metadata.Inputs))
			for i := 0; i < len(resp.Inputs); i++ {
				if metadata.Inputs[i].GetTxIo() != nil {
					resp.Inputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_TxIo{
							TxIo: &pb.IOMetadata_TxIO{
								Address: metadata.Inputs[i].GetTxIo().Address,
								Amount:  metadata.Inputs[i].GetTxIo().Amount,
							},
						},
					}
				} else {
					resp.Inputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_Unknown_{
							Unknown: &pb.IOMetadata_Unknown{},
						},
					}
				}
			}
			resp.Outputs = make([]*pb.IOMetadata, len(metadata.Outputs))
			for i := 0; i < len(resp.Outputs); i++ {
				if metadata.Outputs[i].GetTxIo() != nil {
					resp.Outputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_TxIo{
							TxIo: &pb.IOMetadata_TxIO{
								Address: metadata.Outputs[i].GetTxIo().Address,
								Amount:  metadata.Outputs[i].GetTxIo().Amount,
							},
						},
					}
				} else {
					resp.Outputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_Unknown_{
							Unknown: &pb.IOMetadata_Unknown{},
						},
					}
				}
			}
		}
	}
	return resp, nil
}

// GetAddressTransactions returns a list of transactions for the given address
// Note: only public address are indexed
//
// **Requires AddrIndex**
func (s *GrpcServer) GetAddressTransactions(ctx context.Context, req *pb.GetAddressTransactionsRequest) (*pb.GetAddressTransactionsResponse, error) {
	if s.addrIndex == nil {
		return nil, status.Error(codes.Unavailable, "addr index is not available")
	}

	addr, err := walletlib.DecodeAddress(req.Address, s.chainParams)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "malformed address")
	}
	txids, err := s.addrIndex.GetTransactionsIDs(s.ds, addr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	n := len(txids)
	if req.NbFetch > 0 && int(req.NbFetch) < n {
		n = int(req.NbFetch)
	}
	txs := make([]*pb.GetAddressTransactionsResponse_TransactionWithMetadata, 0, n)
	for i := int(req.NbSkip); i < len(txids); i++ {
		tx, blockID, err := s.txIndex.GetTransaction(s.ds, txids[i])
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		height, err := s.chain.GetBlockHeight(blockID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		txwm := &pb.GetAddressTransactionsResponse_TransactionWithMetadata{
			Tx:       tx,
			Block_ID: blockID.Bytes(),
			Height:   height,
		}

		metadata, err := s.addrIndex.GetTransactionMetadata(s.ds, tx.ID())
		if err != nil && !errors.Is(err, datastore.ErrNotFound) {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if errors.Is(err, datastore.ErrNotFound) {
			txwm.Inputs = make([]*pb.IOMetadata, len(tx.Nullifiers()))
			for i := 0; i < len(txwm.Inputs); i++ {
				txwm.Inputs[i] = &pb.IOMetadata{
					IoType: &pb.IOMetadata_Unknown_{
						Unknown: &pb.IOMetadata_Unknown{},
					},
				}
			}
			txwm.Outputs = make([]*pb.IOMetadata, len(tx.Outputs()))
			for i := 0; i < len(txwm.Outputs); i++ {
				txwm.Outputs[i] = &pb.IOMetadata{
					IoType: &pb.IOMetadata_Unknown_{
						Unknown: &pb.IOMetadata_Unknown{},
					},
				}
			}
		} else {
			txwm.Inputs = make([]*pb.IOMetadata, len(metadata.Inputs))
			for i := 0; i < len(txwm.Inputs); i++ {
				if metadata.Inputs[i].GetTxIo() != nil {
					txwm.Inputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_TxIo{
							TxIo: &pb.IOMetadata_TxIO{
								Address: metadata.Inputs[i].GetTxIo().Address,
								Amount:  metadata.Inputs[i].GetTxIo().Amount,
							},
						},
					}
				} else {
					txwm.Inputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_Unknown_{
							Unknown: &pb.IOMetadata_Unknown{},
						},
					}
				}
			}
			txwm.Outputs = make([]*pb.IOMetadata, len(metadata.Outputs))
			for i := 0; i < len(txwm.Outputs); i++ {
				if metadata.Outputs[i].GetTxIo() != nil {
					txwm.Outputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_TxIo{
							TxIo: &pb.IOMetadata_TxIO{
								Address: metadata.Outputs[i].GetTxIo().Address,
								Amount:  metadata.Outputs[i].GetTxIo().Amount,
							},
						},
					}
				} else {
					txwm.Outputs[i] = &pb.IOMetadata{
						IoType: &pb.IOMetadata_Unknown_{
							Unknown: &pb.IOMetadata_Unknown{},
						},
					}
				}
			}
		}

		txs = append(txs, txwm)
		if len(txs) >= n {
			break
		}
	}
	return &pb.GetAddressTransactionsResponse{
		Txs: txs,
	}, nil
}

// GetMerkleProof returns a Merkle (SPV) proof for a specific transaction
// in the provided block.
//
// **Requires TxIndex**
func (s *GrpcServer) GetMerkleProof(ctx context.Context, req *pb.GetMerkleProofRequest) (*pb.GetMerkleProofResponse, error) {
	if s.txIndex == nil {
		return nil, status.Error(codes.Unavailable, "tx index is not available")
	}
	tx, _, err := s.txIndex.GetTransaction(s.ds, types.NewID(req.Transaction_ID))
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
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

	merkles := blockchain.BuildMerkleTreeStore(blk.Txids())
	hashes, flags := blockchain.MerkleInclusionProof(merkles, tx.ID())
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
			StakeWeight:    uint64(validator.WeightedStake),
			UnclaimedCoins: uint64(validator.UnclaimedCoins),
			EpochBlocks:    validator.EpochBlocks,
		},
	}
	for nullifier, stake := range validator.Nullifiers {
		resp.Validator.Stake = append(resp.Validator.Stake, &pb.Validator_Stake{
			Nullifier:          nullifier[:],
			Amount:             uint64(stake.Amount),
			TimelockedUntil:    stake.Locktime.Unix(),
			Expiration:         stake.Blockstamp.Add(blockchain.ValidatorExpiration).Unix(),
			RestakeEligibility: stake.Blockstamp.Add(blockchain.ValidatorExpiration).Add(-blockchain.RestakePeriod).Unix(),
		})
	}
	return resp, nil
}

// GetValidatorSetInfo returns information about the validator set.
func (s *GrpcServer) GetValidatorSetInfo(ctx context.Context, req *pb.GetValidatorSetInfoRequest) (*pb.GetValidatorSetInfoResponse, error) {
	return &pb.GetValidatorSetInfoResponse{
		TotalStaked:   uint64(s.chain.TotalStaked()),
		StakeWeight:   uint64(s.chain.TotalStakeWeight()),
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
			StakeWeight:    uint64(v.WeightedStake),
			UnclaimedCoins: uint64(v.UnclaimedCoins),
			EpochBlocks:    v.EpochBlocks,
		}
		for nullifier, stake := range v.Nullifiers {
			val.Stake = append(val.Stake, &pb.Validator_Stake{
				Nullifier:          nullifier[:],
				Amount:             uint64(stake.Amount),
				TimelockedUntil:    stake.Locktime.Unix(),
				Expiration:         stake.Blockstamp.Add(blockchain.ValidatorExpiration).Unix(),
				RestakeEligibility: stake.Blockstamp.Add(blockchain.ValidatorExpiration).Add(-blockchain.RestakePeriod).Unix(),
			})
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
	defer sub.Close()

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

// SubscribeCompressedBlocks returns a stream of CompressedBlock notifications when new
// blocks are finalized and connected to the chain.
func (s *GrpcServer) SubscribeCompressedBlocks(req *pb.SubscribeCompressedBlocksRequest, stream pb.BlockchainService_SubscribeCompressedBlocksServer) error {
	sub := s.subscribeEvents()
	defer sub.Close()

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

					txs := make([]*blocks.CompressedBlock_CompressedTx, 0, len(blk.Transactions))
					for _, tx := range blk.Transactions {
						nullifers := make([][]byte, 0, len(tx.Nullifiers()))
						for _, n := range tx.Nullifiers() {
							nullifers = append(nullifers, n.Bytes())
						}
						txs = append(txs, &blocks.CompressedBlock_CompressedTx{
							Txid:       tx.ID().Bytes(),
							Nullifiers: nullifers,
							Outputs:    tx.Outputs(),
						})
					}

					resp := &pb.CompressedBlockNotification{
						Block: &blocks.CompressedBlock{
							Height: blk.Header.Height,
							Txs:    txs,
						},
					}
					if err := stream.Send(resp); err != nil {
						return err
					}
				}
			}
		}
	}
}
