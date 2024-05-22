// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/tidwall/sjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"strconv"
	"time"
)

type GetMempoolInfo struct {
	opts *options
}

func (x *GetMempoolInfo) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetMempoolInfo(makeContext(x.opts.AuthToken), &pb.GetMempoolInfoRequest{})
	if err != nil {
		return err
	}

	m := protojson.MarshalOptions{
		Indent:          "    ",
		EmitUnpopulated: true,
	}
	out, err := m.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetMempool struct {
	opts *options
}

func (x *GetMempool) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetMempool(makeContext(x.opts.AuthToken), &pb.GetMempoolRequest{
		FullTransactions: false,
	})
	if err != nil {
		return err
	}

	ids := make([]types.HexEncodable, 0, len(resp.TransactionData))
	for _, txid := range resp.TransactionData {
		ids = append(ids, txid.GetTransaction_ID())
	}
	out, err := json.MarshalIndent(ids, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetBlockchainInfo struct {
	opts *options
}

func (x *GetBlockchainInfo) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetBlockchainInfo(makeContext(x.opts.AuthToken), &pb.GetBlockchainInfoRequest{})
	if err != nil {
		return err
	}

	s := struct {
		Network           string             `json:"network"`
		BestHeight        uint32             `json:"bestHeight"`
		BestBlockID       types.HexEncodable `json:"bestBlockID"`
		BlockTime         time.Time          `json:"blockTime"`
		TxIndex           bool               `json:"txIndex"`
		CirculatingSupply types.Amount       `json:"circulatingSupply"`
		TotalStaked       types.Amount       `json:"totalStaked"`
		TreasuryBalance   types.Amount       `json:"treasuryBalance"`
		BlockchainSize    uint64             `json:"blockchainSize"`
		Epoch             uint32             `json:"epoch"`
	}{
		Network:           resp.Network.String(),
		BestHeight:        resp.BestHeight,
		BestBlockID:       resp.BestBlock_ID,
		BlockTime:         time.Unix(resp.BlockTime, 0),
		TxIndex:           resp.TxIndex,
		CirculatingSupply: types.Amount(resp.CirculatingSupply),
		TotalStaked:       types.Amount(resp.TotalStaked),
		TreasuryBalance:   types.Amount(resp.TreasuryBalance),
		BlockchainSize:    resp.BlockchainSize,
		Epoch:             resp.Epoch,
	}

	out, err := json.MarshalIndent(&s, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type GetBlockInfo struct {
	opts    *options
	BlockID string `short:"i" long:"id" description:"Block ID to look up. Either use this or the height."`
	Height  int    `short:"t" long:"height" description:"Block height. Either use this or the ID"`
}

func (x *GetBlockInfo) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	var req *pb.GetBlockInfoRequest
	if x.BlockID != "" {
		b, err := hex.DecodeString(x.BlockID)
		if err != nil {
			return err
		}
		req = &pb.GetBlockInfoRequest{
			IdOrHeight: &pb.GetBlockInfoRequest_Block_ID{Block_ID: b},
		}
	} else {
		req = &pb.GetBlockInfoRequest{
			IdOrHeight: &pb.GetBlockInfoRequest_Height{Height: uint32(x.Height)},
		}
	}
	resp, err := client.GetBlockInfo(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}

	m := protojson.MarshalOptions{
		Indent:          "    ",
		EmitUnpopulated: true,
	}
	out, err := m.Marshal(resp.Info)
	if err != nil {
		return err
	}

	value, err := sjson.Set(string(out), "blockID", hex.EncodeToString(resp.Info.Block_ID))
	if err != nil {
		return err
	}
	value, err = sjson.Set(value, "parent", hex.EncodeToString(resp.Info.Parent))
	if err != nil {
		return err
	}
	value, err = sjson.Set(value, "txRoot", hex.EncodeToString(resp.Info.TxRoot))
	if err != nil {
		return err
	}
	value, err = sjson.Set(value, "child", hex.EncodeToString(resp.Info.Child))
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

type GetBlock struct {
	opts    *options
	BlockID string `short:"i" long:"id" description:"Block ID to look up. Either use this or the height."`
	Height  int    `short:"t" long:"height" description:"Block height. Either use this or the ID"`
}

func (x *GetBlock) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	var req *pb.GetRawBlockRequest
	if x.BlockID != "" {
		b, err := hex.DecodeString(x.BlockID)
		if err != nil {
			return err
		}
		req = &pb.GetRawBlockRequest{
			IdOrHeight: &pb.GetRawBlockRequest_Block_ID{Block_ID: b},
		}
	} else {
		req = &pb.GetRawBlockRequest{
			IdOrHeight: &pb.GetRawBlockRequest_Height{Height: uint32(x.Height)},
		}
	}
	resp, err := client.GetRawBlock(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}

	txids := make([]types.HexEncodable, 0, len(resp.Block.Transactions))
	for _, tx := range resp.Block.Transactions {
		id := tx.ID()
		txids = append(txids, id[:])
	}
	b := struct {
		Header *blocks.BlockHeader
		Txids  []types.HexEncodable
	}{
		Header: resp.Block.Header,
		Txids:  txids,
	}

	out, err := json.MarshalIndent(&b, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetCompressedBlock struct {
	opts    *options
	BlockID string `short:"i" long:"id" description:"Block ID to look up. Either use this or the height."`
	Height  int    `short:"t" long:"height" description:"Block height. Either use this or the ID"`
}

func (x *GetCompressedBlock) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	var req *pb.GetCompressedBlockRequest
	if x.BlockID != "" {
		b, err := hex.DecodeString(x.BlockID)
		if err != nil {
			return err
		}
		req = &pb.GetCompressedBlockRequest{
			IdOrHeight: &pb.GetCompressedBlockRequest_Block_ID{Block_ID: b},
		}
	} else {
		req = &pb.GetCompressedBlockRequest{
			IdOrHeight: &pb.GetCompressedBlockRequest_Height{Height: uint32(x.Height)},
		}
	}
	resp, err := client.GetCompressedBlock(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(resp.Block, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetTransaction struct {
	Txid    string `short:"i" long:"id" description:"Txid to look up"`
	Concise bool   `short:"c" long:"concise" description:"Return the transaction without the proof"`
	opts    *options
}

func (x *GetTransaction) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}

	txid, err := hex.DecodeString(x.Txid)
	if err != nil {
		return err
	}

	resp, err := client.GetTransaction(makeContext(x.opts.AuthToken), &pb.GetTransactionRequest{
		Transaction_ID: txid,
	})
	if err != nil {
		return err
	}

	if x.Concise {
		switch tx := resp.Tx.GetTx().(type) {
		case *transactions.Transaction_StandardTransaction:
			tx.StandardTransaction.Proof = nil
		case *transactions.Transaction_MintTransaction:
			tx.MintTransaction.Proof = nil
		case *transactions.Transaction_StakeTransaction:
			tx.StakeTransaction.Proof = nil
		case *transactions.Transaction_CoinbaseTransaction:
			tx.CoinbaseTransaction.Proof = nil
		case *transactions.Transaction_TreasuryTransaction:
			tx.TreasuryTransaction.Proof = nil
		}
	}

	out, err := json.MarshalIndent(resp.Tx, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetMerkleProof struct {
	Txid string `short:"i" long:"id" description:"Txid to get the proof for"`
	opts *options
}

func (x *GetMerkleProof) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}

	txid, err := hex.DecodeString(x.Txid)
	if err != nil {
		return err
	}

	resp, err := client.GetMerkleProof(makeContext(x.opts.AuthToken), &pb.GetMerkleProofRequest{
		Transaction_ID: txid,
	})
	if err != nil {
		return err
	}

	hashes := make([]types.HexEncodable, 0, len(resp.Hashes))
	for _, h := range resp.Hashes {
		hashes = append(hashes, h)
	}
	b := struct {
		BlockID types.HexEncodable   `json:"BlockID"`
		Root    types.HexEncodable   `json:"Root"`
		Hashes  []types.HexEncodable `json:"Hashes"`
		Flags   string               `json:"Flags"`
	}{
		BlockID: resp.Block.Block_ID,
		Root:    resp.Block.TxRoot,
		Hashes:  hashes,
		Flags:   strconv.FormatInt(int64(resp.Flags), 2),
	}

	out, err := json.MarshalIndent(&b, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type GetValidator struct {
	ValID string `short:"i" long:"id" description:"Validator ID to look up"`
	opts  *options
}

func (x *GetValidator) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}

	pid, err := peer.Decode(x.ValID)
	if err != nil {
		return err
	}
	pBytes, err := pid.Marshal()
	if err != nil {
		return err
	}
	resp, err := client.GetValidator(makeContext(x.opts.AuthToken), &pb.GetValidatorRequest{
		Validator_ID: pBytes,
	})
	if err != nil {
		return err
	}

	respID, err := peer.IDFromBytes(resp.Validator.Validator_ID)
	if err != nil {
		return err
	}

	type stake struct {
		Nullifier          types.HexEncodable `json:"nullifier"`
		Amount             types.Amount       `json:"amount"`
		TimelockedUntil    *time.Time         `json:"timelockedUntil,omitempty"`
		Expiration         time.Time          `json:"expiration"`
		RestakeEligibility time.Time          `json:"restakeEligibility"`
	}

	stk := make([]*stake, 0, len(resp.Validator.Stake))
	for _, s := range resp.Validator.Stake {
		a := &stake{
			Nullifier:          s.Nullifier,
			Amount:             types.Amount(s.Amount),
			Expiration:         time.Unix(s.Expiration, 0),
			RestakeEligibility: time.Unix(s.RestakeEligibility, 0),
		}
		if s.TimelockedUntil > 0 {
			ts := time.Unix(s.TimelockedUntil, 0)
			a.TimelockedUntil = &ts
		}
		stk = append(stk, a)
	}

	v := struct {
		ValidatorID    string       `json:"validatorID"`
		TotalStake     types.Amount `json:"totalStake"`
		StakeWeight    types.Amount `json:"stakeWeight"`
		UnclaimedCoins types.Amount `json:"unclaimedCoins"`
		EpochBlocks    uint32       `json:"epochBlocks"`
		ValidatorSince time.Time    `json:"validatorSince"`
		Stake          []*stake     `json:"stake"`
	}{
		ValidatorID:    respID.String(),
		TotalStake:     types.Amount(resp.Validator.TotalStake),
		StakeWeight:    types.Amount(resp.Validator.StakeWeight),
		Stake:          stk,
		UnclaimedCoins: types.Amount(resp.Validator.UnclaimedCoins),
		EpochBlocks:    resp.Validator.EpochBlocks,
		ValidatorSince: time.Unix(resp.Validator.ValidatorSince, 0),
	}

	out, err := json.MarshalIndent(&v, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetValidatorSetInfo struct {
	opts *options
}

func (x *GetValidatorSetInfo) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetValidatorSetInfo(makeContext(x.opts.AuthToken), &pb.GetValidatorSetInfoRequest{})
	if err != nil {
		return err
	}

	s := struct {
		TotalStaked   types.Amount `json:"totalStaked"`
		StakeWeight   types.Amount `json:"stakeWeight"`
		NumValidators uint32       `json:"numValidators"`
	}{
		TotalStaked:   types.Amount(resp.TotalStaked),
		StakeWeight:   types.Amount(resp.StakeWeight),
		NumValidators: resp.NumValidators,
	}

	out, err := json.MarshalIndent(&s, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type GetValidatorSet struct {
	opts *options
}

func (x *GetValidatorSet) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetValidatorSet(makeContext(x.opts.AuthToken), &pb.GetValidatorSetRequest{})
	if err != nil {
		return err
	}

	type stake struct {
		Nullifier          types.HexEncodable `json:"nullifier"`
		Amount             types.Amount       `json:"amount"`
		TimelockedUntil    *time.Time         `json:"timelockedUntil,omitempty"`
		Expiration         time.Time          `json:"expiration"`
		RestakeEligibility time.Time          `json:"restakeEligibility"`
	}

	type v struct {
		ValidatorID    string       `json:"validatorID"`
		TotalStake     types.Amount `json:"totalStake"`
		StakeWeight    types.Amount `json:"stakeWeight"`
		UnclaimedCoins types.Amount `json:"unclaimedCoins"`
		EpochBlocks    uint32       `json:"epochBlocks"`
		Stake          []*stake     `json:"stake"`
	}
	vals := make([]v, 0, len(resp.Validators))
	for _, val := range resp.Validators {
		respID, err := peer.IDFromBytes(val.Validator_ID)
		if err != nil {
			return err
		}
		stk := make([]*stake, 0, len(val.Stake))
		for _, s := range val.Stake {
			a := &stake{
				Nullifier:          s.Nullifier,
				Amount:             types.Amount(s.Amount),
				Expiration:         time.Unix(s.Expiration, 0),
				RestakeEligibility: time.Unix(s.RestakeEligibility, 0),
			}
			if s.TimelockedUntil > 0 {
				ts := time.Unix(s.TimelockedUntil, 0)
				a.TimelockedUntil = &ts
			}
			stk = append(stk, a)
		}
		vals = append(vals, v{
			ValidatorID:    respID.String(),
			TotalStake:     types.Amount(val.TotalStake),
			StakeWeight:    types.Amount(val.StakeWeight),
			Stake:          stk,
			UnclaimedCoins: types.Amount(val.UnclaimedCoins),
			EpochBlocks:    val.EpochBlocks,
		})
	}
	out, err := json.MarshalIndent(vals, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetAccumulatorCheckpoint struct {
	opts      *options
	Timestamp int64  `short:"s" long:"timestamp" description:"A timestamp either at or after the desired checkpoint. Use either this or height."`
	Height    uint32 `short:"t" long:"height" description:"A block height either at or after the desired checkpoint. Use either this or timestamp."`
}

func (x *GetAccumulatorCheckpoint) Execute(args []string) error {
	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}
	var req *pb.GetAccumulatorCheckpointRequest
	if x.Timestamp > 0 {
		req = &pb.GetAccumulatorCheckpointRequest{
			HeightOrTimestamp: &pb.GetAccumulatorCheckpointRequest_Timestamp{
				Timestamp: x.Timestamp,
			},
		}
	} else {
		req = &pb.GetAccumulatorCheckpointRequest{
			HeightOrTimestamp: &pb.GetAccumulatorCheckpointRequest_Height{
				Height: x.Height,
			},
		}
	}

	resp, err := client.GetAccumulatorCheckpoint(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}

	acc := make([]types.HexEncodable, 0, len(resp.Accumulator))
	for _, a := range resp.Accumulator {
		acc = append(acc, a)
	}

	r := struct {
		Height      uint32
		Accumulator []types.HexEncodable
		NumEntries  uint64
	}{
		Height:      resp.Height,
		Accumulator: acc,
		NumEntries:  resp.NumEntries,
	}

	out, err := json.MarshalIndent(&r, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type SubmitTransaction struct {
	opts *options
	Tx   string `short:"t" long:"tx" description:"The transaction to submit. Serialized as hex string or JSON."`
}

func (x *SubmitTransaction) Execute(args []string) error {
	var tx transactions.Transaction
	txBytes, err := hex.DecodeString(x.Tx)
	if err == nil {
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return err
		}
	} else {
		if err := json.Unmarshal([]byte(x.Tx), &tx); err != nil {
			return err
		}
	}

	client, err := makeBlockchainClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.SubmitTransaction(makeContext(x.opts.AuthToken), &pb.SubmitTransactionRequest{
		Transaction: &tx,
	})
	if err != nil {
		return err
	}

	fmt.Println(hex.EncodeToString(resp.Transaction_ID))
	return nil
}
