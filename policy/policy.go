// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package policy

import (
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"sync"
)

type Policy struct {
	minFeePerKilobyte  types.Amount
	minStake           types.Amount
	blocksizeSoftLimit uint32
	treasuryWhitelist  []types.ID

	mtx sync.RWMutex
}

func NewPolicy(minFeePerKilobyte, minStake types.Amount, blocksizeSoftLimit uint32) *Policy {
	return &Policy{
		minFeePerKilobyte:  minFeePerKilobyte,
		minStake:           minStake,
		blocksizeSoftLimit: blocksizeSoftLimit,
		treasuryWhitelist:  nil,
		mtx:                sync.RWMutex{},
	}
}

func (p *Policy) GetMinFeePerKilobyte() types.Amount {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.minFeePerKilobyte
}

func (p *Policy) SetMinFeePerKilobyte(fpkb types.Amount) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.minFeePerKilobyte = fpkb
}

func (p *Policy) GetMinStake() types.Amount {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.minStake
}

func (p *Policy) SetMinStake(amt types.Amount) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.minStake = amt
}

func (p *Policy) GetBlocksizeSoftLimit() uint32 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.blocksizeSoftLimit
}

func (p *Policy) SetBlocksizeSoftLimit(limit uint32) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.blocksizeSoftLimit = limit
}

func (p *Policy) GetTreasuryWhitelist() []types.ID {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	ret := make([]types.ID, 0, len(p.treasuryWhitelist))
	for _, txid := range p.treasuryWhitelist {
		ret = append(ret, txid.Clone())
	}
	return ret
}

func (p *Policy) AddToTreasuryWhitelist(txids ...types.ID) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.treasuryWhitelist = append(p.treasuryWhitelist, txids...)
}

func (p *Policy) RemoveFromTreasuryWhitelist(txids ...types.ID) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

loop:
	for i := len(p.treasuryWhitelist) - 1; i >= 0; i-- {
		for _, txid := range txids {
			if p.treasuryWhitelist[i] == txid {
				p.treasuryWhitelist = append(p.treasuryWhitelist[:i], p.treasuryWhitelist[i+1:]...)
				continue loop
			}
		}
	}
}

func (p *Policy) IsAcceptableBlock(blk *blocks.Block) (bool, error) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	blockSize, err := blk.SerializedSize()
	if err != nil {
		return false, err
	}
	if uint32(blockSize) > p.blocksizeSoftLimit {
		return false, nil
	}
	for _, tx := range blk.Transactions {
		fpkb, isFeePayer, err := mempool.CalcFeePerKilobyte(tx)
		if err != nil {
			return false, err
		}
		if isFeePayer && fpkb < p.minFeePerKilobyte {
			return false, nil
		}
		switch typ := tx.GetTx().(type) {
		case *transactions.Transaction_TreasuryTransaction:
			exists := false
			treasuryID := typ.TreasuryTransaction.ID()
			for _, txid := range p.treasuryWhitelist {
				if txid == treasuryID {
					exists = true
					break
				}
			}
			if !exists {
				return false, nil
			}
		case *transactions.Transaction_StakeTransaction:
			if types.Amount(typ.StakeTransaction.Amount) < p.minStake {
				return false, nil
			}
		}
	}
	return true, nil
}
