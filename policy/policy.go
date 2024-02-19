// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package policy

import (
	"context"
	"errors"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"path"
	"sync"
)

type Policy struct {
	ds                 repo.Datastore
	minFeePerKilobyte  types.Amount
	minStake           types.Amount
	blocksizeSoftLimit uint32
	treasuryWhitelist  []types.ID

	mtx sync.RWMutex
}

func NewPolicy(ds repo.Datastore, minFeePerKilobyte, minStake types.Amount, blocksizeSoftLimit uint32) (*Policy, error) {
	var treasuryWhitelist []types.ID
	if ds != nil {
		query, err := ds.Query(context.Background(), query.Query{
			Prefix: repo.TreasuryWhitelistDatastoreKeyPrefix,
		})
		if err != nil && !errors.Is(err, datastore.ErrNotFound) {
			return nil, err
		}

		defer query.Close()

		for r := range query.Next() {
			idHex := path.Base(r.Key)
			id, err := types.NewIDFromString(idHex)
			if err != nil {
				return nil, err
			}
			treasuryWhitelist = append(treasuryWhitelist, id)
		}
	}

	return &Policy{
		ds:                 ds,
		minFeePerKilobyte:  minFeePerKilobyte,
		minStake:           minStake,
		blocksizeSoftLimit: blocksizeSoftLimit,
		treasuryWhitelist:  treasuryWhitelist,
		mtx:                sync.RWMutex{},
	}, nil
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

	if p.ds != nil {
		for _, txid := range txids {
			if err := p.ds.Put(context.Background(), datastore.NewKey(repo.TreasuryWhitelistDatastoreKeyPrefix+txid.String()), nil); err != nil {
				log.WithCaller(true).Error("Error putting treasury whitelist to datastore", log.Args("Error", err.Error()))
			}
		}
	}

	p.treasuryWhitelist = append(p.treasuryWhitelist, txids...)
}

func (p *Policy) RemoveFromTreasuryWhitelist(txids ...types.ID) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

loop:
	for i := len(p.treasuryWhitelist) - 1; i >= 0; i-- {
		for _, txid := range txids {
			if p.treasuryWhitelist[i] == txid {
				if p.ds != nil {
					if err := p.ds.Delete(context.Background(), datastore.NewKey(repo.TreasuryWhitelistDatastoreKeyPrefix+txid.String())); err != nil {
						log.WithCaller(true).Error("Error deleting treasury whitelist from datastore", log.Args("Error", err.Error()))
					}
				}
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
		fpkb, isFeePayer, err := CalcFeePerKilobyte(tx)
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

func CalcFeePerKilobyte(tx *transactions.Transaction) (types.Amount, bool, error) {
	var fee uint64
	switch t := tx.GetTx().(type) {
	case *transactions.Transaction_CoinbaseTransaction,
		*transactions.Transaction_TreasuryTransaction,
		*transactions.Transaction_StakeTransaction:
		return 0, false, nil
	case *transactions.Transaction_StandardTransaction:
		fee = t.StandardTransaction.Fee
	case *transactions.Transaction_MintTransaction:
		fee = t.MintTransaction.Fee
	}

	size, err := tx.SerializedSize()
	if err != nil {
		return 0, false, err
	}
	kbs := float64(size) / 1000

	return types.Amount(float64(fee) / kbs), true, nil
}
