// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package protocol

import (
	"context"
	ctxio "github.com/jbenet/go-context/io"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	policy2 "github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/wire"
	"google.golang.org/protobuf/proto"
	"io"
	"time"
)

const (
	PolicyProtocol        = "/policy/"
	PolicyProtocolVersion = "1.0.0"
)

// PolicyService is a libp2p network protocol that allows other peers
// to query this node for its policy preferences.
type PolicyService struct {
	ctx     context.Context
	network *net.Network
	policy  *policy2.Policy
	ms      net.MessageSender
}

// NewPolicyService returns a new PolicyService and starts the stream handler.
func NewPolicyService(ctx context.Context, network *net.Network, params *params.NetworkParams, policy *policy2.Policy) *PolicyService {
	ps := &PolicyService{
		ctx:     ctx,
		network: network,
		policy:  policy,
		ms:      net.NewMessageSender(network.Host(), params.ProtocolPrefix+PolicyProtocol+PolicyProtocolVersion),
	}

	ps.network.Host().SetStreamHandler(params.ProtocolPrefix+PolicyProtocol+PolicyProtocolVersion, ps.HandleNewStream)
	return ps
}

func (ps *PolicyService) HandleNewStream(s inet.Stream) {
	go ps.handleNewMessage(s)
}

func (ps *PolicyService) handleNewMessage(s inet.Stream) {
	defer s.Close()
	contextReader := ctxio.NewReader(ps.ctx, s)
	reader := msgio.NewVarintReaderSize(contextReader, 1<<23)
	remotePeer := s.Conn().RemotePeer()
	defer reader.Close()
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-ticker.C:
			return
		default:
		}

		msgBytes, err := reader.ReadMsg()
		if err != nil {
			reader.ReleaseMsg(msgBytes)
			if err == io.EOF || err == inet.ErrReset {
				s.Close()
				return
			}
			log.Debug("Error reading from policy service stream", log.ArgsFromMap(map[string]any{
				"peer":  remotePeer,
				"error": err,
			}))
			s.Reset()
			return
		}
		req := new(wire.MsgPolicyRequest)
		if err := proto.Unmarshal(msgBytes, req); err != nil {
			reader.ReleaseMsg(msgBytes)
			log.Debug("Error unmarshalling policy service message", log.ArgsFromMap(map[string]any{
				"peer":  remotePeer,
				"error": err,
			}))
			s.Reset()
			return
		}
		reader.ReleaseMsg(msgBytes)

		// Increase the transient banscore so peers don't slam us with
		// crawls. There is really no reason to make their queries in
		// rapid succession.
		ps.network.IncreaseBanscore(remotePeer, 0, 10, "rate limiting policy protocol")

		var resp proto.Message
		switch m := req.Msg.(type) {
		case *wire.MsgPolicyRequest_GetFeePerKb:
			resp, err = ps.handleGetFeePerKb(m.GetFeePerKb)
		case *wire.MsgPolicyRequest_GetMinStake:
			resp, err = ps.handleGetMinStake(m.GetMinStake)
		case *wire.MsgPolicyRequest_GetBlocksizeSoftLimit:
			resp, err = ps.handleGetBlocksizeSoftLimit(m.GetBlocksizeSoftLimit)
		case *wire.MsgPolicyRequest_GetTreasuryWhitelist:
			resp, err = ps.handleGetTreasuryWhitelist(m.GetTreasuryWhitelist)
		default:
			log.WithCaller(true).Error("Received unknown policy message", log.ArgsFromMap(map[string]any{
				"peer": remotePeer,
			}))
			s.Reset()
			ps.network.IncreaseBanscore(remotePeer, 30, 0, "sent unknown policy protocol message")
			return
		}
		if err != nil {
			log.WithCaller(true).Error("Error handling policy service message", log.ArgsFromMap(map[string]any{
				"peer":  remotePeer,
				"error": err,
			}))
			continue
		}

		if resp != nil {
			if err := net.WriteMsg(s, resp); err != nil {
				log.WithCaller(true).Error("Error writing policy service response", log.ArgsFromMap(map[string]any{
					"peer":  remotePeer,
					"error": err,
				}))
				s.Reset()
				return
			}
		}
		ticker.Reset(time.Minute)
	}
}

// GetFeePerKb queries the remote peer for their fee per kb policy
func (ps *PolicyService) GetFeePerKb(p peer.ID) (types.Amount, error) {
	var (
		req = &wire.MsgPolicyRequest{
			Msg: &wire.MsgPolicyRequest_GetFeePerKb{},
		}
		resp = new(wire.MsgGetFeePerKBResp)
	)
	err := ps.ms.SendRequest(ps.ctx, p, req, resp)
	if err != nil {
		return 0, err
	}

	return types.Amount(resp.FeePerKb), nil
}

func (ps *PolicyService) handleGetFeePerKb(req *wire.GetFeePerKB) (*wire.MsgGetFeePerKBResp, error) {
	resp := &wire.MsgGetFeePerKBResp{
		FeePerKb: uint64(ps.policy.GetMinFeePerKilobyte()),
	}
	return resp, nil
}

// GetMinStake queries the remote peer for their minimum stake policy
func (ps *PolicyService) GetMinStake(p peer.ID) (types.Amount, error) {
	var (
		req = &wire.MsgPolicyRequest{
			Msg: &wire.MsgPolicyRequest_GetMinStake{},
		}
		resp = new(wire.MsgGetMinStakeResp)
	)
	err := ps.ms.SendRequest(ps.ctx, p, req, resp)
	if err != nil {
		return 0, err
	}

	return types.Amount(resp.MinStake), nil
}

func (ps *PolicyService) handleGetMinStake(req *wire.GetMinStake) (*wire.MsgGetMinStakeResp, error) {
	resp := &wire.MsgGetMinStakeResp{
		MinStake: uint64(ps.policy.GetMinStake()),
	}
	return resp, nil
}

// GetBlocksizeSoftLimit queries the remote peer for their blocksize soft limit policy
func (ps *PolicyService) GetBlocksizeSoftLimit(p peer.ID) (uint32, error) {
	var (
		req = &wire.MsgPolicyRequest{
			Msg: &wire.MsgPolicyRequest_GetBlocksizeSoftLimit{},
		}
		resp = new(wire.MsgGetBlocksizeSoftLimitResp)
	)
	err := ps.ms.SendRequest(ps.ctx, p, req, resp)
	if err != nil {
		return 0, err
	}

	return resp.Limit, nil
}

func (ps *PolicyService) handleGetBlocksizeSoftLimit(req *wire.GetBlocksizeSoftLimit) (*wire.MsgGetBlocksizeSoftLimitResp, error) {
	resp := &wire.MsgGetBlocksizeSoftLimitResp{
		Limit: ps.policy.GetBlocksizeSoftLimit(),
	}
	return resp, nil
}

// GetTreasuryWhitelist queries the remote peer for their treasury whitelist
func (ps *PolicyService) GetTreasuryWhitelist(p peer.ID) ([]types.ID, error) {
	var (
		req = &wire.MsgPolicyRequest{
			Msg: &wire.MsgPolicyRequest_GetTreasuryWhitelist{},
		}
		resp = new(wire.MsgGetTreasuryWhitelistResp)
	)
	err := ps.ms.SendRequest(ps.ctx, p, req, resp)
	if err != nil {
		return nil, err
	}

	txids := make([]types.ID, 0, len(resp.Whitelist))
	for _, txid := range resp.Whitelist {
		txids = append(txids, types.NewID(txid))
	}
	return txids, nil
}

func (ps *PolicyService) handleGetTreasuryWhitelist(req *wire.GetTreasuryWhitelist) (*wire.MsgGetTreasuryWhitelistResp, error) {
	txids := ps.policy.GetTreasuryWhitelist()
	whitelist := make([][]byte, 0, len(txids))
	for _, txid := range txids {
		whitelist = append(whitelist, txid.Bytes())
	}

	resp := &wire.MsgGetTreasuryWhitelistResp{
		Whitelist: whitelist,
	}
	return resp, nil
}
