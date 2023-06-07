// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"runtime"
	"sync"
)

// TransactionScanner is used to scan transaction outputs and attempt to decrypt
// each one. This allows us to flag outputs to be protected by the accumulator.
// One could perform this function outside the blockchain package and independently
// transaction the accumulator and inclusion proofs, but that would require double
// hashes of the accuumulator for every block.
type TransactionScanner struct {
	keys       []*crypto.Curve25519PrivateKey
	workChan   chan *transactions.Output
	resultChan chan []byte
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
func (s *TransactionScanner) ScanOutputs(outputs []*transactions.Output) map[types.ID]bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	maxGoRoutines := runtime.NumCPU() * 3
	if maxGoRoutines <= 0 {
		maxGoRoutines = 1
	}
	if maxGoRoutines > len(outputs) {
		maxGoRoutines = len(outputs)
	}

	for i := 0; i < maxGoRoutines; i++ {
		go s.scanHandler()
	}

	s.done = make(chan struct{})
	s.workChan = make(chan *transactions.Output)
	s.resultChan = make(chan []byte)

	defer close(s.done)
	defer close(s.resultChan)

	go func() {
		for _, out := range outputs {
			s.workChan <- out
		}
		close(s.workChan)
	}()

	ret := make(map[types.ID]bool)
	for i := 0; i < len(outputs); i++ {
		match := <-s.resultChan
		if match != nil {
			ret[types.NewID(match)] = true
		}
	}
	return ret
}

func (s *TransactionScanner) scanHandler() {
	for {
		select {
		case o := <-s.workChan:
			if o != nil {
				for _, k := range s.keys {
					_, err := k.Decrypt(o.Ciphertext)
					if err == nil {
						s.resultChan <- o.Commitment
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
