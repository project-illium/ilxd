// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"runtime"
	"time"
)

// ValidateTransactionProof validates the zero knowledge proof for a single transaction.
// proofCache must not be nil. The validator will check whether the proof already exists
// in the cache. If it does the proof will be assumed to be valid. If not it will
// validate the proof and add the proof to the cache if valid.
func ValidateTransactionProof(tx *transactions.Transaction, proofCache *ProofCache, verifier zk.Verifier) <-chan error {
	errChan := make(chan error)
	go func() {
		validator := NewProofValidator(proofCache, verifier)
		errChan <- validator.Validate([]*transactions.Transaction{tx})
		close(errChan)
	}()
	return errChan
}

// proofValidator is used to validate transaction zero knowledge proofs in parallel.
type proofValidator struct {
	proofCache *ProofCache
	verifier   zk.Verifier
	workChan   chan *transactions.Transaction
	resultChan chan error
	done       chan struct{}
}

// NewProofValidator returns a new ProofValidator.
// The proofCache must NOT be nil.
func NewProofValidator(proofCache *ProofCache, verifier zk.Verifier) *proofValidator {
	return &proofValidator{
		proofCache: proofCache,
		verifier:   verifier,
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
				outputs := make([]circparams.PublicOutput, 0, len(tx.StandardTransaction.Outputs))
				for _, out := range tx.StandardTransaction.Outputs {
					outputs = append(outputs, circparams.PublicOutput{
						Commitment: types.NewID(out.Commitment),
						CipherText: out.Ciphertext,
					})
				}
				nullifiers := make([]types.Nullifier, 0, len(tx.StandardTransaction.Nullifiers))
				for _, n := range tx.StandardTransaction.Nullifiers {
					nullifiers = append(nullifiers, types.NewNullifier(n))
				}
				params := circparams.PublicParams{
					SigHash:    types.NewID(sigHash),
					Nullifiers: nullifiers,
					TXORoot:    types.NewID(tx.StandardTransaction.TxoRoot),
					Fee:        types.Amount(tx.StandardTransaction.Fee),
					Outputs:    outputs,
				}
				if tx.StandardTransaction.Locktime != nil {
					params.Locktime = time.Unix(tx.StandardTransaction.Locktime.Timestamp, 0)
					params.LocktimePrecision = time.Duration(tx.StandardTransaction.Locktime.Precision) * time.Second
				}
				valid, err := p.verifier.Verify(zk.StandardValidationProgram(), &params, tx.StandardTransaction.Proof)
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
				outputs := make([]circparams.PublicOutput, 0, len(tx.CoinbaseTransaction.Outputs))
				for _, out := range tx.CoinbaseTransaction.Outputs {
					outputs = append(outputs, circparams.PublicOutput{
						Commitment: types.NewID(out.Commitment),
						CipherText: out.Ciphertext,
					})
				}
				params := circparams.PublicParams{
					Coinbase: types.Amount(tx.CoinbaseTransaction.NewCoins),
					Outputs:  outputs,
				}
				valid, err := p.verifier.Verify(zk.CoinbaseValidationProgram(), &params, tx.CoinbaseTransaction.Proof)
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
				outputs := make([]circparams.PublicOutput, 0, len(tx.TreasuryTransaction.Outputs))
				for _, out := range tx.TreasuryTransaction.Outputs {
					outputs = append(outputs, circparams.PublicOutput{
						Commitment: types.NewID(out.Commitment),
						CipherText: out.Ciphertext,
					})
				}
				params := circparams.PublicParams{
					Coinbase: types.Amount(tx.TreasuryTransaction.Amount),
					Outputs:  outputs,
				}
				valid, err := p.verifier.Verify(zk.TreasuryValidationProgram(), &params, tx.TreasuryTransaction.Proof)
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
				outputs := make([]circparams.PublicOutput, 0, len(tx.MintTransaction.Outputs))
				for _, out := range tx.MintTransaction.Outputs {
					outputs = append(outputs, circparams.PublicOutput{
						Commitment: types.NewID(out.Commitment),
						CipherText: out.Ciphertext,
					})
				}
				nullifiers := make([]types.Nullifier, 0, len(tx.MintTransaction.Nullifiers))
				for _, n := range tx.MintTransaction.Nullifiers {
					nullifiers = append(nullifiers, types.NewNullifier(n))
				}
				params := circparams.PublicParams{
					SigHash:    types.NewID(sigHash),
					Nullifiers: nullifiers,
					TXORoot:    types.NewID(tx.MintTransaction.TxoRoot),
					Fee:        types.Amount(tx.MintTransaction.Fee),
					MintID:     types.NewID(tx.MintTransaction.Asset_ID),
					MintAmount: types.Amount(tx.MintTransaction.NewTokens),
					Outputs:    outputs,
				}
				if tx.MintTransaction.Locktime != nil {
					params.Locktime = time.Unix(tx.MintTransaction.Locktime.Timestamp, 0)
					params.LocktimePrecision = time.Duration(tx.MintTransaction.Locktime.Precision) * time.Second
				}
				valid, err := p.verifier.Verify(zk.MintValidationProgram(), &params, tx.MintTransaction.Proof)
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
				params := circparams.StakePublicParams{
					StakeAmount: types.Amount(tx.StakeTransaction.Amount),
					PublicParams: circparams.PublicParams{
						SigHash:    types.NewID(sigHash),
						Nullifiers: []types.Nullifier{types.NewNullifier(tx.StakeTransaction.Nullifier)},
						TXORoot:    types.NewID(tx.StakeTransaction.TxoRoot),
						Locktime:   time.Unix(tx.StakeTransaction.LockedUntil, 0),
					},
				}
				valid, err := p.verifier.Verify(zk.StakeValidationProgram(), &params, tx.StakeTransaction.Proof)
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
