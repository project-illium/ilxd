// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"context"
	"errors"
	"fmt"
	ctxio "github.com/jbenet/go-context/io"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/types/wire"
	"google.golang.org/protobuf/proto"
	"io"
	"time"
)

const (
	ChainServiceProtocol        = "/chainservice/"
	ChainServiceProtocolVersion = "1.0.0"

	maxBatchSize = 2000
)

var ErrNotCurrent = errors.New("peer not current")
var ErrNotFound = errors.New("not found")

type FetchBlockFunc func(blockID types.ID) (*blocks.Block, error)

type ChainService struct {
	ctx        context.Context
	network    *net.Network
	params     *params.NetworkParams
	fetchBlock FetchBlockFunc
	chain      *blockchain.Blockchain
	ms         net.MessageSender
}

func NewChainService(ctx context.Context, fetchBlock FetchBlockFunc, chain *blockchain.Blockchain, network *net.Network, params *params.NetworkParams) (*ChainService, error) {
	cs := &ChainService{
		ctx:        ctx,
		network:    network,
		fetchBlock: fetchBlock,
		chain:      chain,
		params:     params,
		ms:         net.NewMessageSender(network.Host(), params.ProtocolPrefix+ChainServiceProtocol+ChainServiceProtocolVersion),
	}
	pruned, err := chain.IsPruned()
	if err != nil {
		return nil, err
	}
	if !pruned {
		cs.network.Host().SetStreamHandler(cs.params.ProtocolPrefix+ChainServiceProtocol+ChainServiceProtocolVersion, cs.HandleNewStream)
	}
	return cs, nil
}

func (cs *ChainService) HandleNewStream(s inet.Stream) {
	go cs.handleNewMessage(s)
}

func (cs *ChainService) handleNewMessage(s inet.Stream) {
	defer s.Close()
	contextReader := ctxio.NewReader(cs.ctx, s)
	reader := msgio.NewVarintReaderSize(contextReader, 1<<23)
	remotePeer := s.Conn().RemotePeer()
	defer reader.Close()
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-cs.ctx.Done():
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
			log.Debugf("Error reading from chain service stream: peer: %s, error: %s", remotePeer, err.Error())
			s.Reset()
			return
		}
		req := new(wire.MsgChainServiceRequest)
		if err := proto.Unmarshal(msgBytes, req); err != nil {
			reader.ReleaseMsg(msgBytes)
			log.Debugf("Error unmarshalling chain service message: peer: %s, error: %s", remotePeer, err.Error())
			s.Reset()
			return
		}
		reader.ReleaseMsg(msgBytes)

		var resp proto.Message
		switch m := req.Msg.(type) {
		case *wire.MsgChainServiceRequest_GetBlockTxs:
			resp, err = cs.handleGetBlockTxs(m.GetBlockTxs)
		case *wire.MsgChainServiceRequest_GetBlockTxids:
			resp, err = cs.handleGetBlockTxids(m.GetBlockTxids)
		case *wire.MsgChainServiceRequest_GetBlock:
			resp, err = cs.handleGetBlock(m.GetBlock)
		case *wire.MsgChainServiceRequest_GetBlockId:
			resp, err = cs.handleGetBlockID(m.GetBlockId)
		case *wire.MsgChainServiceRequest_GetBest:
			resp, err = cs.handleGetBest(m.GetBest)
		case *wire.MsgChainServiceRequest_GetHeadersStream:
			err = cs.handleGetHeadersStream(m.GetHeadersStream, s)
			if err != nil {
				log.Errorf("Error sending header response to peer: %s, error: %s", remotePeer, err.Error())
				s.Reset()
				return
			}
		case *wire.MsgChainServiceRequest_GetBlockTxsStream:
			err = cs.handleGetBlockTxsStream(m.GetBlockTxsStream, s)
			if err != nil {
				log.Errorf("Error sending block txs response to peer: %s, error: %s", remotePeer, err.Error())
				s.Reset()
				return
			}
		}
		if err != nil {
			log.Errorf("Error handing chain service message to peer: %s, error: %s", remotePeer, err.Error())
			continue
		}

		if resp != nil {
			if err := net.WriteMsg(s, resp); err != nil {
				log.Errorf("Error writing chain service response to peer: %s, error: %s", remotePeer, err.Error())
				s.Reset()
				return
			}
		}
		ticker.Reset(time.Minute)
	}
}

func (cs *ChainService) GetBlockTxs(p peer.ID, blockID types.ID, txIndexes []uint32) ([]*transactions.Transaction, error) {
	var (
		req = &wire.MsgChainServiceRequest{
			Msg: &wire.MsgChainServiceRequest_GetBlockTxs{
				GetBlockTxs: &wire.GetBlockTxsReq{
					Block_ID:  blockID[:],
					TxIndexes: txIndexes,
				},
			},
		}
		resp = new(wire.MsgBlockTxsResp)
	)
	err := cs.ms.SendRequest(cs.ctx, p, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != wire.ErrorResponse_None {
		return nil, fmt.Errorf("error response from peer: %s", resp.GetError().String())
	}

	if len(resp.Transactions) != len(txIndexes) {
		cs.network.IncreaseBanscore(p, 50, 0)
		return nil, fmt.Errorf("peer %s did not return all requested txs", p.String())
	}

	return resp.Transactions, nil
}

func (cs *ChainService) handleGetBlockTxs(req *wire.GetBlockTxsReq) (*wire.MsgBlockTxsResp, error) {
	blk, err := cs.fetchBlock(types.NewID(req.Block_ID))
	if err != nil {
		return &wire.MsgBlockTxsResp{Error: wire.ErrorResponse_NotFound}, nil
	}

	resp := &wire.MsgBlockTxsResp{
		Transactions: make([]*transactions.Transaction, len(req.TxIndexes)),
	}

	for i, idx := range req.TxIndexes {
		if idx > uint32(len(blk.Transactions))-1 {
			return &wire.MsgBlockTxsResp{Error: wire.ErrorResponse_BadRequest}, nil
		}
		resp.Transactions[i] = blk.Transactions[idx]
	}

	return resp, nil
}

func (cs *ChainService) GetBlockTxids(p peer.ID, blockID types.ID) ([]types.ID, error) {
	var (
		req = &wire.MsgChainServiceRequest{
			Msg: &wire.MsgChainServiceRequest_GetBlockTxids{
				GetBlockTxids: &wire.GetBlockTxidsReq{
					Block_ID: blockID[:],
				},
			},
		}
		resp = new(wire.MsgBlockTxidsResp)
	)
	err := cs.ms.SendRequest(cs.ctx, p, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != wire.ErrorResponse_None {
		return nil, fmt.Errorf("error response from peer: %s", resp.GetError().String())
	}

	txids := make([]types.ID, 0, len(resp.Txids))
	for _, txid := range resp.Txids {
		txids = append(txids, types.NewID(txid))
	}

	return txids, nil
}

func (cs *ChainService) handleGetBlockTxids(req *wire.GetBlockTxidsReq) (*wire.MsgBlockTxidsResp, error) {
	blk, err := cs.fetchBlock(types.NewID(req.Block_ID))
	if err != nil {
		return &wire.MsgBlockTxidsResp{Error: wire.ErrorResponse_NotFound}, nil
	}

	txids := make([][]byte, 0, len(blk.Transactions))
	for _, tx := range blk.Transactions {
		id := tx.ID()
		txids = append(txids, id[:])
	}

	resp := &wire.MsgBlockTxidsResp{
		Txids: txids,
	}

	return resp, nil
}

func (cs *ChainService) GetBlock(p peer.ID, blockID types.ID) (*blocks.Block, error) {
	var (
		req = &wire.MsgChainServiceRequest{
			Msg: &wire.MsgChainServiceRequest_GetBlock{
				GetBlock: &wire.GetBlockReq{
					Block_ID: blockID[:],
				},
			},
		}
		resp = new(wire.MsgBlockResp)
	)
	err := cs.ms.SendRequest(cs.ctx, p, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != wire.ErrorResponse_None {
		return nil, fmt.Errorf("error response from peer: %s", resp.GetError().String())
	}

	if resp.Block.ID().Compare(blockID) != 0 {
		return nil, errors.New("incorrect block returned")
	}

	return resp.Block, nil
}

func (cs *ChainService) handleGetBlock(req *wire.GetBlockReq) (*wire.MsgBlockResp, error) {
	blk, err := cs.fetchBlock(types.NewID(req.Block_ID))
	if err != nil {
		return &wire.MsgBlockResp{Error: wire.ErrorResponse_NotFound}, nil
	}

	resp := &wire.MsgBlockResp{
		Block: blk,
	}

	return resp, nil
}

func (cs *ChainService) GetBlockID(p peer.ID, height uint32) (types.ID, error) {
	var (
		req = &wire.MsgChainServiceRequest{
			Msg: &wire.MsgChainServiceRequest_GetBlockId{
				GetBlockId: &wire.GetBlockIDReq{
					Height: height,
				},
			},
		}
		resp = new(wire.MsgGetBlockIDResp)
	)
	err := cs.ms.SendRequest(cs.ctx, p, req, resp)
	if err != nil {
		return types.ID{}, err
	}

	if resp.Error == wire.ErrorResponse_NotFound {
		return types.ID{}, ErrNotFound
	}

	if resp.Error != wire.ErrorResponse_None {
		return types.ID{}, fmt.Errorf("error response from peer: %s", resp.GetError().String())
	}

	return types.NewID(resp.Block_ID), nil
}

func (cs *ChainService) handleGetBlockID(req *wire.GetBlockIDReq) (*wire.MsgGetBlockIDResp, error) {
	blockID, err := cs.chain.GetBlockIDByHeight(req.Height)
	if err != nil {
		return &wire.MsgGetBlockIDResp{Error: wire.ErrorResponse_NotFound}, nil
	}

	resp := &wire.MsgGetBlockIDResp{
		Block_ID: blockID[:],
	}

	return resp, nil
}

func (cs *ChainService) GetHeadersStream(p peer.ID, startHeight uint32) (<-chan *blocks.BlockHeader, error) {
	req := &wire.MsgChainServiceRequest{
		Msg: &wire.MsgChainServiceRequest_GetHeadersStream{
			GetHeadersStream: &wire.GetHeadersStreamReq{
				StartHeight: startHeight,
			},
		},
	}

	s, err := cs.network.Host().NewStream(context.Background(), p, cs.params.ProtocolPrefix+ChainServiceProtocol+ChainServiceProtocolVersion)
	if err != nil {
		return nil, err
	}
	err = net.WriteMsg(s, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *blocks.BlockHeader)

	go func() {
		reader := msgio.NewVarintReaderSize(s, 1<<23)
		for {
			header := new(blocks.BlockHeader)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			if err := net.ReadMsg(ctx, reader, header); err != nil {
				close(ch)
				s.Close()
				cancel()
				return
			}
			ch <- header
		}
	}()

	return ch, nil
}

func (cs *ChainService) handleGetHeadersStream(req *wire.GetHeadersStreamReq, s inet.Stream) error {
	_, bestHeight, _ := cs.chain.BestBlock()

	endHeight := req.StartHeight + maxBatchSize - 1
	if endHeight > bestHeight {
		endHeight = bestHeight
	}

	for i := req.StartHeight; i <= endHeight; i++ {
		header, err := cs.chain.GetHeaderByHeight(i)
		if err != nil {
			s.Close()
			return err
		}
		if err := net.WriteMsg(s, header); err != nil {
			s.Close()
			return err
		}
	}
	return s.Close()
}

func (cs *ChainService) GetBlockTxsStream(p peer.ID, startHeight uint32) (<-chan *blocks.BlockTxs, error) {
	req := &wire.MsgChainServiceRequest{
		Msg: &wire.MsgChainServiceRequest_GetBlockTxsStream{
			GetBlockTxsStream: &wire.GetBlockTxsStreamReq{
				StartHeight: startHeight,
			},
		},
	}

	s, err := cs.network.Host().NewStream(context.Background(), p, cs.params.ProtocolPrefix+ChainServiceProtocol+ChainServiceProtocolVersion)
	if err != nil {
		return nil, err
	}
	err = net.WriteMsg(s, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *blocks.BlockTxs)

	go func() {
		reader := msgio.NewVarintReaderSize(s, 1<<23)
		for {
			txs := new(blocks.BlockTxs)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			if err := net.ReadMsg(ctx, reader, txs); err != nil {
				close(ch)
				s.Close()
				cancel()
				return
			}
			ch <- txs
		}
	}()

	return ch, nil
}

func (cs *ChainService) handleGetBlockTxsStream(req *wire.GetBlockTxsStreamReq, s inet.Stream) error {
	_, bestHeight, _ := cs.chain.BestBlock()

	endHeight := req.StartHeight + maxBatchSize - 1
	if endHeight > bestHeight {
		endHeight = bestHeight
	}

	for i := req.StartHeight; i <= endHeight; i++ {
		block, err := cs.chain.GetBlockByHeight(i)
		if err != nil {
			return err
		}
		if err := net.WriteMsg(s, &blocks.BlockTxs{Transactions: block.Transactions}); err != nil {
			s.Close()
			return err
		}
	}
	return s.Close()
}

func (cs *ChainService) GetBest(p peer.ID) (types.ID, uint32, error) {
	var (
		req = &wire.MsgChainServiceRequest{
			Msg: &wire.MsgChainServiceRequest_GetBest{
				GetBest: &wire.GetBestReq{},
			},
		}
		resp = new(wire.MsgGetBestResp)
	)
	err := cs.ms.SendRequest(cs.ctx, p, req, resp)
	if err != nil {
		return types.ID{}, 0, err
	}

	if resp.Error == wire.ErrorResponse_NotCurrent {
		return types.ID{}, 0, ErrNotCurrent
	}

	if resp.Error != wire.ErrorResponse_None {
		return types.ID{}, 0, fmt.Errorf("error response from peer: %s", resp.GetError().String())
	}

	return types.NewID(resp.Block_ID), resp.Height, nil
}

func (cs *ChainService) handleGetBest(req *wire.GetBestReq) (*wire.MsgGetBestResp, error) {
	blockID, height, _ := cs.chain.BestBlock()

	resp := &wire.MsgGetBestResp{
		Block_ID: blockID[:],
		Height:   height,
	}

	// FIXME: if not current return error

	return resp, nil
}
