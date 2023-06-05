// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/tidwall/sjson"
	"google.golang.org/protobuf/encoding/protojson"
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

	m := protojson.MarshalOptions{
		Indent:          "    ",
		EmitUnpopulated: true,
	}
	out, err := m.Marshal(resp)
	if err != nil {
		return err
	}

	value, err := sjson.Set(string(out), "bestBlockID", hex.EncodeToString(resp.BestBlock_ID))
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

type GetBlockInfo struct {
	opts    *options
	BlockID string `short:"i" long:"id" description:"Block ID to look up. Either us this or the height."`
	Height  int    `short:"t" long:"height" description:"Block height. Either us this or the ID"`
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
