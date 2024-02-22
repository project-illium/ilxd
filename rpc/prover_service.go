// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxProverSteps = 500000

// Prove creates the proof for the transaction and returns the transaction
// with the proof attached. The transaction is *not* submitted to the network.
//
// The transaction is validated against the mempool and will return an error
// if it is an otherwise invalid transaction.
func (s *GrpcServer) Prove(ctx context.Context, req *pb.ProveRequest) (*pb.ProveResponse, error) {
	proof, err := s.proveTx(req.Transaction, req.Inputs, req.Outputs)
	if err != nil {
		return nil, err
	}
	req.Transaction.GetStandardTransaction().Proof = proof
	return &pb.ProveResponse{
		Transaction: req.Transaction,
	}, nil
}

// ProveAndSubmit creates the proof for the transaction and then submits it to
// the network. And error is returned if it fails mempool submission.
func (s *GrpcServer) ProveAndSubmit(ctx context.Context, req *pb.ProveAndSubmitRequest) (*pb.ProveAndSubmitResponse, error) {
	proof, err := s.proveTx(req.Transaction, req.Inputs, req.Outputs)
	if err != nil {
		return nil, err
	}
	req.Transaction.GetStandardTransaction().Proof = proof

	err = s.txMemPool.ProcessTransaction(req.Transaction)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.ProveAndSubmitResponse{
		Transaction_ID: req.Transaction.ID().Bytes(),
	}, nil
}

func (s *GrpcServer) proveTx(tx *transactions.Transaction, privIns []*pb.PrivateInput, privOuts []*pb.PrivateOutput) ([]byte, error) {
	err := s.txMemPool.ValidateTransaction(tx, false)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if tx.GetStandardTransaction() == nil {
		return nil, status.Error(codes.InvalidArgument, "prover service can only prove standard transactions")
	}

	privParams := &circparams.StandardPrivateParams{
		Inputs:  make([]circparams.PrivateInput, len(privIns)),
		Outputs: make([]circparams.PrivateOutput, len(privOuts)),
	}

	for i, in := range privIns {
		if in.TxoProof == nil {
			return nil, status.Error(codes.InvalidArgument, "input txo proof is nil")
		}
		privParams.Inputs[i] = circparams.PrivateInput{
			Amount:          types.Amount(in.Amount),
			AssetID:         types.NewID(in.Asset_ID),
			Salt:            types.NewID(in.Salt),
			State:           types.State{},
			CommitmentIndex: in.TxoProof.Index,
			InclusionProof: circparams.InclusionProof{
				Hashes: in.TxoProof.Hashes,
				Flags:  in.TxoProof.Flags,
			},
			Script:          in.Script,
			LockingParams:   in.LockingParams,
			UnlockingParams: in.UnlockingParams,
		}
		if err := privParams.Inputs[i].State.Deserialize(in.State); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	for i, out := range privOuts {
		privParams.Outputs[i] = circparams.PrivateOutput{
			ScriptHash: types.NewID(out.ScriptHash),
			Amount:     types.Amount(out.Amount),
			AssetID:    types.NewID(out.Asset_ID),
			Salt:       types.NewID(out.Salt),
			State:      types.State{},
		}
		if err := privParams.Outputs[i].State.Deserialize(out.State); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	publicParams, err := tx.GetStandardTransaction().ToCircuitParams()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.prover.Prove(zk.StandardValidationProgram(), privParams, publicParams, maxProverSteps)
}
