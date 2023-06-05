// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"google.golang.org/protobuf/encoding/protojson"
	"strings"
)

type GetHostInfo struct {
	opts *options
}

func (x *GetHostInfo) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetHostInfo(makeContext(x.opts.AuthToken), &pb.GetHostInfoRequest{})
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

type GetPeers struct {
	opts *options
}

func (x *GetPeers) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetPeers(makeContext(x.opts.AuthToken), &pb.GetPeersRequest{})
	if err != nil {
		return err
	}
	if resp.Peers == nil {
		resp.Peers = []*pb.Peer{}
	}

	out, err := json.MarshalIndent(resp.Peers, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type AddPeer struct {
	Addr string `short:"a" long:"addr" description:"The peer address to add. This must be in multiaddr format and include the p2p field. Ex) /ip4/167.172.126.176/tcp/4001/p2p/12D3KooWHnpVyu9XDeFoAVayqr9hvc9xPqSSHtCSFLEkKgcz5Wro"`
	opts *options
}

func (x *AddPeer) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	_, err = client.AddPeer(makeContext(x.opts.AuthToken), &pb.AddPeerRequest{
		Addr: x.Addr,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type BlockPeer struct {
	Peer string `short:"p" long:"peer" description:"The peer ID to block"`
	opts *options
}

func (x *BlockPeer) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	_, err = client.BlockPeer(makeContext(x.opts.AuthToken), &pb.BlockPeerRequest{
		Peer_ID: x.Peer,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type UnblockPeer struct {
	Peer string `short:"p" long:"peer" description:"The peer ID to unblock"`
	opts *options
}

func (x *UnblockPeer) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	_, err = client.UnblockPeer(makeContext(x.opts.AuthToken), &pb.UnblockPeerRequest{
		Peer_ID: x.Peer,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type SetLogLevel struct {
	Level string `short:"l" long:"level" description:"The log level: [DEBUG, INFO, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY]"`
	opts  *options
}

func (x *SetLogLevel) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	var logLevelSeverity = map[string]pb.SetLogLevelRequest_Level{
		"debug":     pb.SetLogLevelRequest_DEBUG,
		"info":      pb.SetLogLevelRequest_INFO,
		"warning":   pb.SetLogLevelRequest_WARNING,
		"error":     pb.SetLogLevelRequest_ERROR,
		"critical":  pb.SetLogLevelRequest_CRITICAL,
		"alert":     pb.SetLogLevelRequest_ALERT,
		"emergency": pb.SetLogLevelRequest_EMERGENCY,
	}
	level, ok := logLevelSeverity[strings.ToLower(x.Level)]
	if !ok {
		return errors.New("unknown log level")
	}
	_, err = client.SetLogLevel(makeContext(x.opts.AuthToken), &pb.SetLogLevelRequest{
		Level: level,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type GetMinFeePerKilobyte struct {
	opts *options
}

func (x *GetMinFeePerKilobyte) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetMinFeePerKilobyte(makeContext(x.opts.AuthToken), &pb.GetMinFeePerKilobyteRequest{})
	if err != nil {
		return err
	}

	fmt.Println(resp.FeePerKilobyte)
	return nil
}

type SetMinFeePerKilobyte struct {
	Fee  uint64 `short:"f" long:"feeperkb" description:"The fee per kilobyte to set"`
	opts *options
}

func (x *SetMinFeePerKilobyte) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	_, err = client.SetMinFeePerKilobyte(makeContext(x.opts.AuthToken), &pb.SetMinFeePerKilobyteRequest{
		FeePerKilobyte: x.Fee,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type GetMinStake struct {
	opts *options
}

func (x *GetMinStake) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetMinStake(makeContext(x.opts.AuthToken), &pb.GetMinStakeRequest{})
	if err != nil {
		return err
	}

	fmt.Println(resp.MinStakeAmount)
	return nil
}

type SetMinStake struct {
	Amount uint64 `short:"m" long:"minstake" description:"The minimum stake amount to set"`
	opts   *options
}

func (x *SetMinStake) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.SetMinStake(makeContext(x.opts.AuthToken), &pb.SetMinStakeRequest{
		MinStakeAmount: x.Amount,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type GetBlockSizeSoftLimit struct {
	opts *options
}

func (x *GetBlockSizeSoftLimit) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetBlockSizeSoftLimit(makeContext(x.opts.AuthToken), &pb.GetBlockSizeSoftLimitRequest{})
	if err != nil {
		return err
	}

	fmt.Println(resp.BlockSize)
	return nil
}

type SetBlockSizeSoftLimit struct {
	Limit uint32 `short:"l" long:"limit" description:"The blocksize soft limit in bytes"`
	opts  *options
}

func (x *SetBlockSizeSoftLimit) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	_, err = client.SetBlockSizeSoftLimit(makeContext(x.opts.AuthToken), &pb.SetBlockSizeSoftLimitRequest{
		BlockSize: x.Limit,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type GetTreasuryWhitelist struct {
	opts *options
}

func (x *GetTreasuryWhitelist) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetTreasuryWhitelist(makeContext(x.opts.AuthToken), &pb.GetTreasuryWhitelistRequest{})
	if err != nil {
		return err
	}

	ids := make([]types.HexEncodable, 0, len(resp.Txids))
	for _, txid := range resp.Txids {
		ids = append(ids, txid)
	}
	out, err := json.MarshalIndent(ids, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type UpdateTreasuryWhitelist struct {
	ToAdd    []string `short:"a" long:"toadd" description:"A treasury transaction txid to add to the whitelist"`
	ToRemove []string `short:"r" long:"toremove" description:"A treasury transaction txid to remove from the whitelist"`
	opts     *options
}

func (x *UpdateTreasuryWhitelist) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}
	toAdd := make([][]byte, 0, len(x.ToAdd))
	for _, a := range x.ToAdd {
		id, err := hex.DecodeString(a)
		if err != nil {
			return err
		}
		toAdd = append(toAdd, id)
	}
	toRemove := make([][]byte, 0, len(x.ToRemove))
	for _, r := range x.ToRemove {
		id, err := hex.DecodeString(r)
		if err != nil {
			return err
		}
		toRemove = append(toRemove, id)
	}

	_, err = client.UpdateTreasuryWhitelist(makeContext(x.opts.AuthToken), &pb.UpdateTreasuryWhitelistRequest{
		Add:    toAdd,
		Remove: toRemove,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type ReconsiderBlock struct {
	opts    *options
	BlockID string `short:"i" long:"id" description:"Block ID of the block to reconsider"`
	PeerID  string `short:"p" long:"peer" description:"Optional peer to try to download the block from"`
}

func (x *ReconsiderBlock) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}

	blockID, err := hex.DecodeString(x.BlockID)
	if err != nil {
		return err
	}

	var p string
	if x.PeerID != "" {
		pid, err := peer.Decode(x.PeerID)
		if err != nil {
			return err
		}
		p = pid.String()
	}

	_, err = client.ReconsiderBlock(makeContext(x.opts.AuthToken), &pb.ReconsiderBlockRequest{
		Block_ID:     blockID,
		DownloadPeer: p,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type RecomputeChainState struct {
	opts *options
}

func (x *RecomputeChainState) Execute(args []string) error {
	client, err := makeNodeClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.RecomputeChainState(makeContext(x.opts.AuthToken), &pb.RecomputeChainStateRequest{})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}
