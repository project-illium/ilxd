// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
)

type Policy struct {
	MinFeePerByte      types.Amount
	MinStake           types.Amount
	BlocksizeSoftLimit uint32
	MaxMessageSize     uint32
	TreasuryWhitelist  []types.ID
}

func (p *Policy) IsPreferredBlock(blk *blocks.Block) (bool, error) {
	blockSize, err := blk.SerializedSize()
	if err != nil {
		return false, err
	}
	if uint32(blockSize) > p.BlocksizeSoftLimit {
		return false, nil
	}
	for _, tx := range blk.Transactions {
		fpb, isFeePayer, err := mempool.CalcFeePerByte(tx)
		if err != nil {
			return false, err
		}
		if isFeePayer && fpb < p.MinFeePerByte {
			return false, nil
		}
		switch typ := tx.GetTx().(type) {
		case *transactions.Transaction_TreasuryTransaction:
			exists := false
			treasuryID := typ.TreasuryTransaction.ID()
			for _, txid := range p.TreasuryWhitelist {
				if txid == treasuryID {
					exists = true
					break
				}
			}
			if !exists {
				return false, nil
			}
		case *transactions.Transaction_StakeTransaction:
			if types.Amount(typ.StakeTransaction.Amount) < p.MinStake {
				return false, nil
			}
		}
	}
	return true, nil
}
