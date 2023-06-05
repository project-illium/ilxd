// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/tidwall/sjson"
	"google.golang.org/protobuf/encoding/protojson"
)

type GetBlockchainInfo struct{}

func (x *GetBlockchainInfo) Execute(args []string) error {
	client, err := makeBlockchainClient()
	if err != nil {
		return err
	}
	resp, err := client.GetBlockchainInfo(context.Background(), &pb.GetBlockchainInfoRequest{})
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
