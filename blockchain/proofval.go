// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"runtime"
	"time"
)

func ValidateTransactionProof(tx *transactions.Transaction, blockTime time.Time) error {
	validator := NewProofValidator(blockTime)
	return validator.Validate([]*transactions.Transaction{tx})
}

type proofValidator struct {
	workChan   chan *transactions.Transaction
	resultChan chan error
	done       chan struct{}
	blockTime  time.Time
}

func NewProofValidator(blockTime time.Time) *proofValidator {
	return &proofValidator{
		workChan:   make(chan *transactions.Transaction),
		resultChan: make(chan error),
		done:       make(chan struct{}),
		blockTime:  blockTime,
	}
}

func (p *proofValidator) Validate(txs []*transactions.Transaction) error {
	defer close(p.done)

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
				sigHash, err := tx.StandardTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputCommitments := make([][]byte, 0, len(tx.StandardTransaction.Outputs))
				for _, out := range tx.StandardTransaction.Outputs {
					outputCommitments = append(outputCommitments, out.Commitment)
				}
				params := standard.PublicParams{
					TXORoot:           tx.StandardTransaction.TxoRoot,
					SigHash:           sigHash,
					OutputCommitments: outputCommitments,
					Nullifiers:        tx.StandardTransaction.Nullifiers,
					Fee:               tx.StandardTransaction.Fee,
					Coinbase:          0,
					MintID:            nil,
					MintAmount:        0,
					Blocktime:         p.blockTime,
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
				p.resultChan <- nil
			case *transactions.Transaction_CoinbaseTransaction:
				sigHash, err := tx.CoinbaseTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputCommitments := make([][]byte, 0, len(tx.CoinbaseTransaction.Outputs))
				for _, out := range tx.CoinbaseTransaction.Outputs {
					outputCommitments = append(outputCommitments, out.Commitment)
				}
				params := standard.PublicParams{
					TXORoot:           nil,
					SigHash:           sigHash,
					OutputCommitments: outputCommitments,
					Nullifiers:        nil,
					Fee:               0,
					Coinbase:          tx.CoinbaseTransaction.NewCoins,
					MintID:            nil,
					MintAmount:        0,
					Blocktime:         p.blockTime,
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
				p.resultChan <- nil
			case *transactions.Transaction_TreasuryTransaction:
				sigHash, err := tx.TreasuryTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputCommitments := make([][]byte, 0, len(tx.TreasuryTransaction.Outputs))
				for _, out := range tx.TreasuryTransaction.Outputs {
					outputCommitments = append(outputCommitments, out.Commitment)
				}
				params := standard.PublicParams{
					TXORoot:           nil,
					SigHash:           sigHash,
					OutputCommitments: outputCommitments,
					Nullifiers:        nil,
					Fee:               0,
					Coinbase:          tx.TreasuryTransaction.Amount,
					MintID:            nil,
					MintAmount:        0,
					Blocktime:         p.blockTime,
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
				p.resultChan <- nil
			case *transactions.Transaction_MintTransaction:
				sigHash, err := tx.MintTransaction.SigHash()
				if err != nil {
					p.resultChan <- err
					break
				}
				outputCommitments := make([][]byte, 0, len(tx.MintTransaction.Outputs))
				for _, out := range tx.MintTransaction.Outputs {
					outputCommitments = append(outputCommitments, out.Commitment)
				}
				params := standard.PublicParams{
					TXORoot:           tx.MintTransaction.TxoRoot,
					SigHash:           sigHash,
					OutputCommitments: outputCommitments,
					Nullifiers:        tx.MintTransaction.Nullifiers,
					Fee:               tx.MintTransaction.Fee,
					Coinbase:          0,
					MintID:            tx.MintTransaction.Asset_ID,
					MintAmount:        tx.MintTransaction.NewTokens,
					Blocktime:         p.blockTime,
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
				p.resultChan <- nil
			case *transactions.Transaction_StakeTransaction:
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
					Blocktime: p.blockTime,
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
				p.resultChan <- nil
			}
		case <-p.done:
			return
		}
	}
}
