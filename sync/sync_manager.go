// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	nextHeightQuerySize = 8
	bestHeightQuerySize = 100
	lookaheadSize       = 10000
	evaluationWindow    = 5000
)

type SyncManager struct {
	ctx             context.Context
	params          *params.NetworkParams
	network         *net.Network
	chainService    *ChainService
	chain           *blockchain.Blockchain
	consensuChooser ConsensusChooser
	buckets         map[types.ID][]peer.ID
	bucketMtx       sync.RWMutex
	currentMtx      sync.RWMutex
	current         bool
}

type ConsensusChooser func([]*blocks.Block) (types.ID, error)

func NewSyncManager(ctx context.Context, chain *blockchain.Blockchain, network *net.Network, params *params.NetworkParams, cs *ChainService, chooser ConsensusChooser) *SyncManager {
	sm := &SyncManager{
		ctx:             ctx,
		params:          params,
		network:         network,
		chainService:    cs,
		chain:           chain,
		consensuChooser: chooser,
		buckets:         make(map[types.ID][]peer.ID),
		bucketMtx:       sync.RWMutex{},
		currentMtx:      sync.RWMutex{},
	}
	notifier := &inet.NotifyBundle{
		DisconnectedF: sm.bucketPeerDisconnected,
	}

	sm.network.Host().Network().Notify(notifier)

	return sm
}

func (sm *SyncManager) Start() {
	_, startheight, _ := sm.chain.BestBlock()

	// Sync up to the checkpoints if we're not already past them.
	if len(sm.params.Checkpoints) > 0 && startheight < sm.params.Checkpoints[len(sm.params.Checkpoints)-1].Height {
		sm.syncToCheckpoints(startheight)
	}

	// Before we start we want to do a large peer query to see if there
	// are any forks out there. If there are, we will sort the peers into
	// buckets depending on which fork they are on.
	for {
		err := sm.populatePeerBuckets()
		if err != nil {
			select {
			case <-time.After(time.Second * 10):
				continue
			}
		}
		break
	}

	// Now we can continue to sync the rest of the chain.
syncLoop:
	for {
		sm.currentMtx.RLock()
		if sm.current {
			return
		}
		sm.currentMtx.RUnlock()
		// We'll start by querying a subset of our peers ask them for what
		// block they have at height + lookaheadSize.
		//
		// We will make sure at least one peer from each bucket is part of
		// the subset that we query. This will ensure that, if there is a
		// fork, we will encounter it as we sync forward.
		bestID, height, _ := sm.chain.BestBlock()
		blockMap, err := sm.queryPeersForBlockID(height + lookaheadSize)
		if err != nil || len(blockMap) == 0 {
			select {
			case <-time.After(time.Second * 10):
				continue
			}
		}
		if len(blockMap) == 1 {
			// All peers agree on the blockID at the requested height. This is good.
			// We'll just sync up to this height.
			for blockID, p := range blockMap {
				if bestID == blockID {
					sm.currentMtx.Lock()
					defer sm.currentMtx.Unlock()
					sm.current = true
					return
				}
				err := sm.syncBlocks(p, height+1, height+lookaheadSize, bestID, blockID)
				if err != nil {
					log.Debugf("Error syncing blocks. Peer: %s, Err: %s", p, err)
				}
				break
			}
		} else {
			// The peers disagree on the block at the requested height. This sucks.
			// We'll download the evaluation window for each chain and select the one
			// with the best chain score.
			//
			// Step one is we need to find the fork point.
			forkPoint, err := sm.findForkPoint(height, height+lookaheadSize, blockMap)
			if err != nil {
				log.Debugf("Error find fork point. Err: %s", err)
				continue
			}

			// Step two is sync up to fork point.
			if forkPoint > height {
				for blockID, p := range blockMap {
					err := sm.syncBlocks(p, height+1, forkPoint, bestID, blockID)
					if err != nil {
						log.Debugf("Error syncing blocks. Peer: %s, Err: %s", p, err)
						continue syncLoop
					}
					break
				}
			}

			var (
				scores      = make(map[types.ID]blockchain.ChainScore)
				tipOfChain  = true
				firstBlocks = make([]*blocks.Block, 0, len(blockMap))
			)

			// Step three is to download the evaluation window for each side of the fork.
			for blockID, p := range blockMap {
				blks, err := sm.downloadEvalWindow(p, forkPoint+1)
				if err != nil {
					log.Debugf("Sync peer failed to serve evaluation window. Banning. Peer: %s", p)
					sm.network.IncreaseBanscore(p, 101, 0)
					continue syncLoop
				}
				firstBlocks = append(firstBlocks, blks[0])

				// Step four is to compute the chain score for each side of the fork.
				score, err := sm.chain.CalcChainScore(blks)
				if err != nil {
					log.Debugf("Sync peer failed to served invalid evaluation window. Banning. Peer: %s", p)
					sm.network.IncreaseBanscore(p, 101, 0)
					continue syncLoop
				}
				if len(blks) < evaluationWindow {
					score = score / blockchain.ChainScore(len(blks)) * evaluationWindow
				} else {
					tipOfChain = false
				}
				scores[blockID] = score
			}

			// Finally, sync to the fork with the best chain score.
			var (
				bestScore = blockchain.ChainScore(math.MaxInt32)
				bestID    types.ID
			)
			if tipOfChain {
				bestID, err = sm.consensuChooser(firstBlocks)
				if err != nil {
					log.Debugf("Sync error choosing between tips: %s", err)
					continue syncLoop
				}
			} else {
				for blockID, score := range scores {
					if score < bestScore {
						bestScore = score
						bestID = blockID
					}
				}
			}
			for blockID, p := range blockMap {
				if blockID != bestID {
					sm.network.IncreaseBanscore(p, 101, 0)
				}
			}

			currentID, height, _ := sm.chain.BestBlock()
			err = sm.syncBlocks(blockMap[bestID], height+1, height+lookaheadSize, currentID, bestID)
			if err != nil {
				log.Debugf("Error syncing blocks. Peer: %s, Err: %s", blockMap[bestID], err)
				continue syncLoop
			}
		}
	}
}

func (sm *SyncManager) IsCurrent() bool {
	sm.currentMtx.RLock()
	defer sm.currentMtx.RUnlock()

	return sm.current
}

func (sm *SyncManager) SetCurrent() {
	sm.currentMtx.Lock()
	defer sm.currentMtx.Unlock()

	sm.current = true
}

func (sm *SyncManager) bucketPeerDisconnected(_ inet.Network, conn inet.Conn) {
	sm.bucketMtx.Lock()
	defer sm.bucketMtx.Unlock()

	for blockID, bucket := range sm.buckets {
		for i := len(bucket) - 1; i >= 0; i-- {
			if bucket[i] == conn.RemotePeer() {
				sm.buckets[blockID] = append(sm.buckets[blockID][:i], sm.buckets[blockID][i+1:]...)
			}
		}
		if len(bucket) == 0 {
			delete(sm.buckets, blockID)
		}
	}
}

func (sm *SyncManager) queryPeersForBlockID(height uint32) (map[types.ID]peer.ID, error) {
	peers := sm.network.Host().Network().Peers()
	if len(peers) == 0 {
		return nil, errors.New("no peers to query")
	}
	size := nextHeightQuerySize
	if len(peers) < nextHeightQuerySize {
		size = len(peers)
	}

	toQuery := make(map[peer.ID]bool)
	for len(toQuery) < size {
		p := peers[rand.Intn(len(peers))]
		if toQuery[p] {
			continue
		}
		toQuery[p] = true
	}

	// Add a peer from each bucket to make sure that as
	// we're syncing we discover any forks that might be
	// out there.
	sm.bucketMtx.RLock()
bucketLoop:
	for _, bucket := range sm.buckets {
		for _, p := range bucket {
			if toQuery[p] {
				continue bucketLoop
			}
		}
		p := bucket[rand.Intn(len(bucket))]
		toQuery[p] = true
	}
	sm.bucketMtx.RUnlock()

	type resp struct {
		p       peer.ID
		blockID types.ID
	}

	ch := make(chan resp)
	wg := sync.WaitGroup{}
	wg.Add(len(toQuery))
	go func() {
		for p := range toQuery {
			go func(pid peer.ID, w *sync.WaitGroup) {
				defer w.Done()
				id, err := sm.chainService.GetBlockID(p, height)
				if err != nil {
					sm.network.IncreaseBanscore(pid, 0, 20)
					return
				}
				ch <- resp{
					p:       pid,
					blockID: id,
				}
			}(p, &wg)
		}
		wg.Wait()
		close(ch)
	}()
	ret := make(map[types.ID]peer.ID)
	count := 0
	for r := range ch {
		ret[r.blockID] = r.p
		count++
	}
	// If enough peers failed, return error.
	if count < size/2 {
		return nil, errors.New("less than half of peers returned height query response")
	}
	return ret, nil
}

// populatePeerBuckets queries a large number of peers and asks them when their best
// blockID is. If the peers disagree they will be sorted into buckets based on which
// chain they follow.
//
// Note do to the asynchronous nature of the network peers might not report the same
// best blockID even though they are all following the same chain. In this case we
// may still end up putting them into different buckets. This is OK as the buckets
// are only used to add peers to our queries and if there is no fork this won't hurt
// anything.
func (sm *SyncManager) populatePeerBuckets() error {
	peers := sm.network.Host().Network().Peers()
	if len(peers) == 0 {
		return errors.New("no peers to query")
	}
	size := bestHeightQuerySize
	if len(peers) < bestHeightQuerySize {
		size = len(peers)
	}

	toQuery := make(map[peer.ID]bool)
	for len(toQuery) < size {
		p := peers[rand.Intn(len(peers))]
		if toQuery[p] {
			continue
		}
		toQuery[p] = true
	}

	buckets := make(map[types.ID][]peer.ID)

	type resp struct {
		p       peer.ID
		blockID types.ID
		height  uint32
		current bool
	}

	ch := make(chan resp)
	wg := sync.WaitGroup{}
	wg.Add(len(toQuery))
	go func() {
		for p := range toQuery {
			go func(pid peer.ID, w *sync.WaitGroup) {
				defer w.Done()
				id, height, err := sm.chainService.GetBest(p)
				if errors.Is(err, ErrNotCurrent) {
					return
				} else if err != nil {
					sm.network.IncreaseBanscore(pid, 0, 20)
					return
				}
				ch <- resp{
					p:       pid,
					blockID: id,
					height:  height,
				}
			}(p, &wg)
		}
		wg.Wait()
		close(ch)
	}()
	count := 0
	for r := range ch {
		count++
		if _, ok := buckets[r.blockID]; !ok {
			buckets[r.blockID] = make([]peer.ID, 0)
		}
		buckets[r.blockID] = append(buckets[r.blockID], r.p)
	}
	// If enough peers failed, return error.
	if count < size/2 {
		return errors.New("less than half of peers returned height query response")
	}
	sm.buckets = buckets
	return nil
}

func (sm *SyncManager) syncToCheckpoints(currentHeight uint32) {
	startHeight := currentHeight + 1
	parent := sm.params.GenesisBlock.ID()
	for z, checkpoint := range sm.params.Checkpoints {
		if currentHeight > checkpoint.Height {
			continue
		}
		if z > 0 {
			parent = sm.params.Checkpoints[z-1].BlockID
		}
		for {
			peers := sm.network.Host().Network().Peers()
			if len(peers) == 0 {
				select {
				case <-time.After(time.Second * 5):
					continue
				}
			}
			p := peers[rand.Intn(len(peers))]
			err := sm.syncBlocks(p, startHeight, checkpoint.Height, parent, checkpoint.BlockID)
			if err != nil {
				log.Debugf("Error syncing checkpoints. Peer: %s, Err: %s", p, err)
				continue
			}
			break
		}
		startHeight = checkpoint.Height + 1
	}
}

func (sm *SyncManager) downloadEvalWindow(p peer.ID, fromHeight uint32) ([]*blocks.Block, error) {
	headers, err := sm.downloadHeaders(p, fromHeight, fromHeight+evaluationWindow)
	if err != nil {
		sm.network.IncreaseBanscore(p, 0, 20)
		return nil, err
	}
	blks := make([]*blocks.Block, 0, len(headers))
	txs, err := sm.downloadBlockTxs(p, fromHeight, fromHeight+evaluationWindow)
	if err != nil {
		sm.network.IncreaseBanscore(p, 0, 20)
		return nil, fmt.Errorf("peer %s block download error %s", p, err)
	}
	for i, tx := range txs {
		blks = append(blks, &blocks.Block{
			Header:       headers[i],
			Transactions: tx.Transactions,
		})
	}
	return blks, nil
}

func (sm *SyncManager) syncBlocks(p peer.ID, fromHeight, toHeight uint32, parent, expectedID types.ID) error {
	headers, err := sm.downloadHeaders(p, fromHeight, toHeight)
	if err != nil {
		sm.network.IncreaseBanscore(p, 0, 20)
		return err
	}
	if headers[len(headers)-1].ID().Compare(expectedID) != 0 {
		sm.network.IncreaseBanscore(p, 101, 0)
		return fmt.Errorf("peer %s returned last header with unexpected ID", p)
	}

	if types.NewID(headers[0].Parent).Compare(parent) != 0 {
		sm.network.IncreaseBanscore(p, 101, 0)
		return fmt.Errorf("peer %s returned frist header with unexpected parent ID", p)
	}
	for i := len(headers) - 1; i > 0; i-- {
		if types.NewID(headers[i].Parent).Compare(headers[i-1].ID()) != 0 {
			sm.network.IncreaseBanscore(p, 101, 0)
			return fmt.Errorf("peer %s returned headers that do not connect", p)
		}
	}

	var (
		blks      []*blocks.Block
		start     = headers[0].Height
		endHeight = headers[len(headers)-1].Height
		headerIdx = 0
	)

	for {
		blks = make([]*blocks.Block, 0, len(headers))

		stop := start + maxBatchSize
		if stop > endHeight {
			stop = endHeight
		}
		txs, err := sm.downloadBlockTxs(p, start, stop)
		if err != nil {
			sm.network.IncreaseBanscore(p, 0, 20)
			return fmt.Errorf("peer %s block download error %s", p, err)
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
				return fmt.Errorf("peer %s invalid block download merkle root", p.String())
			}
			blks = append(blks, blk)
			x++
		}
		headerIdx += len(txs)
		for _, blk := range blks {
			if err := sm.chain.ConnectBlock(blk, blockchain.BFFastAdd); err != nil {
				return fmt.Errorf("error committing block from peer %s. Height: %d, Err: %s", p, blk.Header.Height, err)
			}
		}
		start = stop + 1
		if stop == endHeight {
			break
		}
	}
	return nil
}

func (sm *SyncManager) findForkPoint(currentHeight, toHeight uint32, blockMap map[types.ID]peer.ID) (uint32, error) {
	type resp struct {
		p       peer.ID
		blockID types.ID
		err     error
	}

	midPoint := (toHeight - currentHeight) / 2
	for {
		ch := make(chan resp)
		wg := sync.WaitGroup{}
		wg.Add(len(blockMap))

		go func(getHeight uint32) {
			for _, p := range blockMap {
				go func(pid peer.ID, w *sync.WaitGroup) {
					defer w.Done()
					id, err := sm.chainService.GetBlockID(p, getHeight)
					ch <- resp{
						p:       pid,
						blockID: id,
						err:     err,
					}
				}(p, &wg)
			}
			wg.Wait()
			close(ch)
		}(midPoint)
		retMap := make(map[types.ID]struct{})
		for r := range ch {
			if r.err != nil {
				return 0, r.err
			}
			retMap[r.blockID] = struct{}{}
		}
		if len(retMap) > 1 {
			midPoint = (midPoint - currentHeight) / 2
			toHeight = midPoint
		} else {
			midPoint = (toHeight - midPoint) / 2
			currentHeight = midPoint
		}
		if toHeight-1 == midPoint {
			return midPoint, nil
		}
	}
}

func (sm *SyncManager) downloadHeaders(p peer.ID, startHeight, endHeight uint32) ([]*blocks.BlockHeader, error) {
	headers := make([]*blocks.BlockHeader, 0, endHeight-startHeight)
	height := startHeight
	for {
		ch, err := sm.chainService.GetHeadersStream(p, height)
		if err != nil {
			return nil, err
		}
		count := 0
		for header := range ch {
			headers = append(headers, header)
			height++
			if height > endHeight {
				return headers, nil
			}
			count++
		}
		if count == 0 {
			return nil, errors.New("peer closed stream without sending any headers")
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
		count := 0
		for blockTxs := range ch {
			txs = append(txs, blockTxs)
			height++
			if height > endHeight {
				return txs, nil
			}
			count++
		}
		if count == 0 {
			return nil, errors.New("peer closed stream without returning any blocktxs")
		}
		if height > endHeight {
			break
		}
	}
	return txs, nil
}
