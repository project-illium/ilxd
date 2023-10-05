// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"runtime"
	"time"
)

// ValidateTransactionProof validates the zero knowledge proof for a single transaction.
// proofCache must not be nil. The validator will check whether the proof already exists
// in the cache. If it does the proof will be assumed to be valid. If not it will
// validate the proof and add the proof to the cache if valid.
func ValidateTransactionProof(tx *transactions.Transaction, proofCache *ProofCache) <-chan error {
	errChan := make(chan error)
	go func() {
		validator := NewProofValidator(proofCache)
		errChan <- validator.Validate([]*transactions.Transaction{tx})
		close(errChan)
	}()
	return errChan
}

// proofValidator is used to validate transaction zero knowledge proofs in parallel.
type proofValidator struct {
	proofCache *ProofCache
	workChan   chan *transactions.Transaction
	resultChan chan error
	done       chan struct{}
}

// NewProofValidator returns a new ProofValidator.
// The proofCache must NOT be nil.
func NewProofValidator(proofCache *ProofCache) *proofValidator {
	return &proofValidator{
		proofCache: proofCache,
		workChan:   make(chan *transactions.Transaction),
		resultChan: make(chan error),
		done:       make(chan struct{}),
	}
}

// Validate validates the transactions proofs in parallel for fast validation.
// If a proof already exists in the proofCache, the validation will be skipped.
// If a proof is valid and does not exist in the cache, it will be added to the
// cache.
func (p *proofValidator) Validate(txs []*transactions.Transaction) error {
	defer close(p.done)

	if len(txs) == 0 {
		return nil
	}

	maxGoRoutines := runtime.NumCPU() * 3
	if maxGoRoutines <= 0 {
		maxGoRoutines = 1
	}
	if maxGoRoutines > len(txs) {
		maxGoRoutines = len(txs)
	}

	for i := 0; i < maxGoRoutines; i++ {
		go p.validateHandler()
	}

	go func() {
		for _, tx := range txs {
			p.workChan <- tx
		}
		close(p.workChan)
	}()

	for i := 0; i < len(txs); i++ {
		err := <-p.resultChan
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *proofValidator) validateHandler() {
	for {
		select {
		case t := <-p.workChan:
			switch tx := t.GetTx().(type) {
			case *transactions.Transaction_StandardTransaction:
				proofHash := types.NewIDFromData(tx.StandardTransaction.Proof)
				exists := p.proofCache.Exists(proofHash, tx.StandardTransaction.Proof, tx.StandardTransaction.ID())
				if exists {
					p.resultChan <- nil
					break
				}

				sigHash, err := tx.StandardTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputs := make([]standard.PublicOutput, 0, len(tx.StandardTransaction.Outputs))
				for _, out := range tx.StandardTransaction.Outputs {
					outputs = append(outputs, standard.PublicOutput{
						Commitment: out.Commitment,
						CipherText: out.Ciphertext,
					})
				}

				params := standard.PublicParams{
					TXORoot:    tx.StandardTransaction.TxoRoot,
					SigHash:    sigHash,
					Outputs:    outputs,
					Nullifiers: tx.StandardTransaction.Nullifiers,
					Fee:        tx.StandardTransaction.Fee,
					Coinbase:   0,
					MintID:     nil,
					MintAmount: 0,
					Locktime:   time.Unix(tx.StandardTransaction.Locktime, 0),
				}

				valid, err := zk.ValidateSnark(standard.StandardCircuit, &params, tx.StandardTransaction.Proof)
				if err != nil {
					p.resultChan <- err
					break
				}
				if !valid {
					p.resultChan <- ruleError(ErrInvalidTx, "invalid zk-snark proof")
					break
				}
				p.proofCache.Add(proofHash, tx.StandardTransaction.Proof, tx.StandardTransaction.ID())
				p.resultChan <- nil
			case *transactions.Transaction_CoinbaseTransaction:
				proofHash := types.NewIDFromData(tx.CoinbaseTransaction.Proof)
				exists := p.proofCache.Exists(proofHash, tx.CoinbaseTransaction.Proof, tx.CoinbaseTransaction.ID())
				if exists {
					p.resultChan <- nil
					break
				}
				sigHash, err := tx.CoinbaseTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputs := make([]standard.PublicOutput, 0, len(tx.CoinbaseTransaction.Outputs))
				for _, out := range tx.CoinbaseTransaction.Outputs {
					outputs = append(outputs, standard.PublicOutput{
						Commitment: out.Commitment,
						CipherText: out.Ciphertext,
					})
				}
				params := standard.PublicParams{
					TXORoot:    nil,
					SigHash:    sigHash,
					Outputs:    outputs,
					Nullifiers: nil,
					Fee:        0,
					Coinbase:   tx.CoinbaseTransaction.NewCoins,
					MintID:     nil,
					MintAmount: 0,
					Locktime:   time.Time{},
				}
				valid, err := zk.ValidateSnark(standard.StandardCircuit, &params, tx.CoinbaseTransaction.Proof)
				if err != nil {
					p.resultChan <- err
					break
				}
				if !valid {
					p.resultChan <- ruleError(ErrInvalidTx, "invalid zk-snark proof")
					break
				}
				p.proofCache.Add(proofHash, tx.CoinbaseTransaction.Proof, tx.CoinbaseTransaction.ID())
				p.resultChan <- nil
			case *transactions.Transaction_TreasuryTransaction:
				proofHash := types.NewIDFromData(tx.TreasuryTransaction.Proof)
				exists := p.proofCache.Exists(proofHash, tx.TreasuryTransaction.Proof, tx.TreasuryTransaction.ID())
				if exists {
					p.resultChan <- nil
					break
				}
				sigHash, err := tx.TreasuryTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputs := make([]standard.PublicOutput, 0, len(tx.TreasuryTransaction.Outputs))
				for _, out := range tx.TreasuryTransaction.Outputs {
					outputs = append(outputs, standard.PublicOutput{
						Commitment: out.Commitment,
						CipherText: out.Ciphertext,
					})
				}
				params := standard.PublicParams{
					TXORoot:    nil,
					SigHash:    sigHash,
					Outputs:    outputs,
					Nullifiers: nil,
					Fee:        0,
					Coinbase:   tx.TreasuryTransaction.Amount,
					MintID:     nil,
					MintAmount: 0,
					Locktime:   time.Time{},
				}
				valid, err := zk.ValidateSnark(standard.StandardCircuit, &params, tx.TreasuryTransaction.Proof)
				if err != nil {
					p.resultChan <- err
					break
				}
				if !valid {
					p.resultChan <- ruleError(ErrInvalidTx, "invalid zk-snark proof")
					break
				}
				p.proofCache.Add(proofHash, tx.TreasuryTransaction.Proof, tx.TreasuryTransaction.ID())
				p.resultChan <- nil
			case *transactions.Transaction_MintTransaction:
				proofHash := types.NewIDFromData(tx.MintTransaction.Proof)
				exists := p.proofCache.Exists(proofHash, tx.MintTransaction.Proof, tx.MintTransaction.ID())
				if exists {
					p.resultChan <- nil
					break
				}
				sigHash, err := tx.MintTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputs := make([]standard.PublicOutput, 0, len(tx.MintTransaction.Outputs))
				for _, out := range tx.MintTransaction.Outputs {
					outputs = append(outputs, standard.PublicOutput{
						Commitment: out.Commitment,
						CipherText: out.Ciphertext,
					})
				}
				params := standard.PublicParams{
					TXORoot:    tx.MintTransaction.TxoRoot,
					SigHash:    sigHash,
					Outputs:    outputs,
					Nullifiers: tx.MintTransaction.Nullifiers,
					Fee:        tx.MintTransaction.Fee,
					Coinbase:   0,
					MintID:     tx.MintTransaction.Asset_ID,
					MintAmount: tx.MintTransaction.NewTokens,
					Locktime:   time.Unix(tx.MintTransaction.Locktime, 0),
				}
				valid, err := zk.ValidateSnark(standard.StandardCircuit, &params, tx.MintTransaction.Proof)
				if err != nil {
					p.resultChan <- err
					break
				}
				if !valid {
					p.resultChan <- ruleError(ErrInvalidTx, "invalid zk-snark proof")
					break
				}
				p.proofCache.Add(proofHash, tx.MintTransaction.Proof, tx.MintTransaction.ID())
				p.resultChan <- nil
			case *transactions.Transaction_StakeTransaction:
				proofHash := types.NewIDFromData(tx.StakeTransaction.Proof)
				exists := p.proofCache.Exists(proofHash, tx.StakeTransaction.Proof, tx.StakeTransaction.ID())
				if exists {
					p.resultChan <- nil
					break
				}
				sigHash, err := tx.StakeTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				params := stake.PublicParams{
					TXORoot:   tx.StakeTransaction.TxoRoot,
					SigHash:   sigHash,
					Amount:    tx.StakeTransaction.Amount,
					Nullifier: tx.StakeTransaction.Nullifier,
					Locktime:  time.Unix(tx.StakeTransaction.Locktime, 0),
				}
				valid, err := zk.ValidateSnark(stake.StakeCircuit, &params, tx.StakeTransaction.Proof)
				if err != nil {
					p.resultChan <- err
					break
				}
				if !valid {
					p.resultChan <- ruleError(ErrInvalidTx, "invalid zk-snark proof")
					break
				}
				p.proofCache.Add(proofHash, tx.StakeTransaction.Proof, tx.StakeTransaction.ID())
				p.resultChan <- nil
			}
		case <-p.done:
			return
		}
	}
}
