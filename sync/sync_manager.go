// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"bytes"
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"math/rand"
)

type SyncManager struct {
	ctx          context.Context
	params       *params.NetworkParams
	network      *net.Network
	chainService *ChainService
	chain        *blockchain.Blockchain
}

func NewSyncManager(ctx context.Context, chain *blockchain.Blockchain, network *net.Network, params *params.NetworkParams, cs *ChainService) *SyncManager {
	return &SyncManager{
		ctx:          ctx,
		params:       params,
		network:      network,
		chainService: cs,
		chain:        chain,
	}
}

func (sm *SyncManager) syncToCheckpoints(currentHeight uint32) error {
	height := currentHeight + 1
	for _, checkpoint := range sm.params.Checkpoints {
		if currentHeight > checkpoint.Height {
			continue
		}
		var (
			headers []*blocks.BlockHeader
			blks    []*blocks.Block
			err     error
		)
		for {
			peers := sm.network.Host().Network().Peers()
			if len(peers) == 0 {
				return errors.New("no peers to sync from")
			}
			p := peers[rand.Intn(len(peers))]
			headers, err = sm.downloadHeaders(p, height, checkpoint.Height)
			if err != nil {
				sm.network.IncreaseBanscore(p, 0, 20)
				continue
			}
			if headers[len(headers)-1].ID().Compare(checkpoint.BlockID) != 0 {
				sm.network.IncreaseBanscore(p, 101, 0)
				continue
			}
			for i := len(headers) - 1; i > 0; i-- {
				if types.NewID(headers[i].Parent).Compare(headers[i-1].ID()) != 0 {
					sm.network.IncreaseBanscore(p, 101, 0)
					continue
				}
			}
			break
		}
		start := headers[0].Height
		endHeight := headers[len(headers)-1].Height
		headerIdx := 0
	blockLoop:
		for {
			blks = make([]*blocks.Block, 0, len(headers))
			peers := sm.network.Host().Network().Peers()
			if len(peers) == 0 {
				return errors.New("no peers to sync from")
			}
			p := peers[rand.Intn(len(peers))]
			stop := start + maxBatchSize
			if stop > endHeight {
				stop = endHeight
			}
			txs, err := sm.downloadBlockTxs(p, start, stop)
			if err != nil {
				sm.network.IncreaseBanscore(p, 0, 20)
				continue
			}
			x := 0
			for i := headerIdx; i < headerIdx+len(txs); i++ {
				blk := &blocks.Block{
					Header:       headers[i],
					Transactions: txs[x].Transactions,
				}
				merkles := blockchain.BuildMerkleTreeStore(blk.Transactions)
				if !bytes.Equal(merkles[len(merkles)-1], headers[i].TxRoot) {
					sm.network.IncreaseBanscore(p, 101, 0)
					continue blockLoop
				}
				blks = append(blks, blk)
				x++
			}
			headerIdx += len(txs)
			for _, blk := range blks {
				if err := sm.chain.ConnectBlock(blk, blockchain.BFFastAdd); err != nil {
					log.Errorf("Error committing checkpointed block. Height: %d, Err: %s", blk.Header.Height, err)
				}
			}
			start = stop + 1
			if stop == endHeight {
				break
			}
		}
		height = checkpoint.Height
	}
	return nil
}

func (sm *SyncManager) downloadHeaders(p peer.ID, startHeight, endHeight uint32) ([]*blocks.BlockHeader, error) {
	headers := make([]*blocks.BlockHeader, 0, endHeight-startHeight)
	height := startHeight
	for {
		ch, err := sm.chainService.GetHeadersStream(p, height)
		if err != nil {
			return nil, err
		}
		for header := range ch {
			headers = append(headers, header)
			height++
			if height > endHeight {
				return headers, nil
			}
		}
		if height > endHeight {
			break
		}
	}
	return headers, nil
}

func (sm *SyncManager) downloadBlockTxs(p peer.ID, startHeight, endHeight uint32) ([]*blocks.BlockTxs, error) {
	txs := make([]*blocks.BlockTxs, 0, endHeight-startHeight)
	height := startHeight
	for {
		ch, err := sm.chainService.GetBlockTxsStream(p, height)
		if err != nil {
			return nil, err
		}
		for blockTxs := range ch {
			txs = append(txs, blockTxs)
			height++
			if height > endHeight {
				return txs, nil
			}
		}
		if height > endHeight {
			break
		}
	}
	return txs, nil
}
