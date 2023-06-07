// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"runtime"
	"sync"
)

// ScanUpdate is used to update subscribes when a new block is connected.
type ScanUpdate struct {
	matches map[types.ID]*ScanMatch
	blk     *blocks.Block
}

// ScanMatch represents an output that has decrypted with one of
// our scan keys.
type ScanMatch struct {
	Key           *crypto.Curve25519PrivateKey
	Commitment    types.ID
	DecryptedNote []byte
	AccIndex      uint64
}

type scanWork struct {
	tx    *transactions.Transaction
	index int
}

// TransactionScanner is used to scan transaction outputs and attempt to decrypt
// each one. This allows us to flag outputs to be protected by the accumulator.
// One could perform this function outside the blockchain package and independently
// transaction the accumulator and inclusion proofs, but that would require double
// hashes of the accuumulator for every block.
type TransactionScanner struct {
	keys       []*crypto.Curve25519PrivateKey
	workChan   chan *scanWork
	resultChan chan *ScanMatch
	done       chan struct{}
	mtx        sync.Mutex
}

// NewTransactionScanner returns a new TransactionScanner
func NewTransactionScanner(keys ...*crypto.Curve25519PrivateKey) *TransactionScanner {
	return &TransactionScanner{
		keys: keys,
		mtx:  sync.Mutex{},
	}
}

// AddKeys adds new scan keys to the TransactionScanner
func (s *TransactionScanner) AddKeys(keys ...*crypto.Curve25519PrivateKey) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.keys = append(s.keys, keys...)
}

// ScanOutputs attempts to decrypt the outputs using the keys and returns a map of matches
func (s *TransactionScanner) ScanOutputs(blk *blocks.Block) map[types.ID]*ScanMatch {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	ret := make(map[types.ID]*ScanMatch)
	if len(s.keys) == 0 {
		return ret
	}

	maxGoRoutines := runtime.NumCPU() * 3
	if maxGoRoutines <= 0 {
		maxGoRoutines = 1
	}
	outputs := len(blk.Outputs())
	if maxGoRoutines > outputs {
		maxGoRoutines = outputs
	}

	for i := 0; i < maxGoRoutines; i++ {
		go s.scanHandler()
	}

	s.done = make(chan struct{})
	s.workChan = make(chan *scanWork)
	s.resultChan = make(chan *ScanMatch)

	defer close(s.done)
	defer close(s.resultChan)

	go func() {
		for _, tx := range blk.Transactions {
			for i := range tx.Outputs() {
				s.workChan <- &scanWork{
					tx:    tx,
					index: i,
				}
			}
		}
		close(s.workChan)
	}()

	for i := 0; i < outputs; i++ {
		match := <-s.resultChan
		if match != nil {
			ret[match.Commitment] = match
		}
	}
	return ret
}

func (s *TransactionScanner) scanHandler() {
	for {
		select {
		case w := <-s.workChan:
			if w != nil {
				for _, k := range s.keys {
					decrypted, err := k.Decrypt(w.tx.Outputs()[w.index].Ciphertext)
					if err == nil {
						s.resultChan <- &ScanMatch{
							Key:           k,
							Commitment:    types.NewID(w.tx.Outputs()[w.index].Commitment),
							DecryptedNote: decrypted,
						}
					} else {
						s.resultChan <- nil
					}
				}
			}
		case <-s.done:
			return
		}
	}
}
