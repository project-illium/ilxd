// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/types/blocks"
	"sync"
)

type Batch struct {
	chain *Blockchain
	blks  []*blocks.Block
	opts  int
	size  int
	wg    sync.WaitGroup
}

func (b *Batch) AddBlock(blk *blocks.Block) error {
	proofVal := NewProofValidator(b.chain.proofCache, b.chain.verifier)
	sigVal := NewSigValidator(b.chain.sigCache)

	b.wg.Add(len(blk.Transactions) * 2)

	go func() {
		proofVal.Validate(blk.Transactions)
		b.wg.Done()
	}()
	go func() {
		sigVal.Validate(blk.Transactions)
		b.wg.Done()
	}()

	ops, size, err := datastoreTxnLimits(blk, bannedNullifiers)
	if err != nil {
		return err
	}
	b.opts += ops
	b.size += size
	b.blks = append(b.blks, blk)
	return nil
}

func (b *Batch) Commit() error {
	b.wg.Wait()
	
	return nil
}
