// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"runtime"
	"time"
)

func ValidateTransactionSig(tx *transactions.Transaction, sigCache *SigCache) error {
	validator := NewSigValidator(sigCache)
	return validator.Validate([]*transactions.Transaction{tx})
}

type sigValidator struct {
	sigCache   *SigCache
	workChan   chan *transactions.Transaction
	resultChan chan error
	done       chan struct{}
	blockTime  time.Time
}

func NewSigValidator(sigCache *SigCache) *sigValidator {
	return &sigValidator{
		sigCache:   sigCache,
		workChan:   make(chan *transactions.Transaction),
		resultChan: make(chan error),
		done:       make(chan struct{}),
	}
}

func (s *sigValidator) Validate(txs []*transactions.Transaction) error {
	defer close(s.done)

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
		go s.validateHandler()
	}

	go func() {
		for _, tx := range txs {
			s.workChan <- tx
		}
		close(s.workChan)
	}()

	for i := 0; i < len(txs); i++ {
		err := <-s.resultChan
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sigValidator) validateHandler() {
	for {
		select {
		case t := <-s.workChan:
			switch tx := t.GetTx().(type) {
			case *transactions.Transaction_CoinbaseTransaction:
				validatorID, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
				if err != nil {
					s.resultChan <- ruleError(ErrInvalidTx, "coinbase tx validator ID does not decode")
					break
				}

				validatorPubkey, err := validatorID.ExtractPublicKey()
				if err != nil {
					s.resultChan <- ruleError(ErrInvalidTx, "coinbase tx validator pubkey invalid")
					break
				}

				sigHash, err := tx.CoinbaseTransaction.SigHash()
				if err != nil {
					s.resultChan <- err
					break
				}

				exists := s.sigCache.Exists(types.NewID(sigHash), tx.CoinbaseTransaction.Signature, validatorPubkey)
				if exists {
					s.resultChan <- nil
					break
				}

				valid, err := validatorPubkey.Verify(sigHash, tx.CoinbaseTransaction.Signature)
				if err != nil {
					s.resultChan <- err
					break
				}
				if !valid {
					s.resultChan <- ruleError(ErrInvalidTx, "coinbase tx invalid signature")
					break
				}
				s.sigCache.Add(types.NewID(sigHash), tx.CoinbaseTransaction.Signature, validatorPubkey)
				s.resultChan <- nil
			case *transactions.Transaction_MintTransaction:
				mintKey, err := crypto.UnmarshalEd25519PublicKey(tx.MintTransaction.MintKey)
				if err != nil {
					s.resultChan <- ruleError(ErrInvalidTx, "mint tx pubkey invalid")
					break
				}

				sigHash, err := tx.MintTransaction.SigHash()
				if err != nil {
					s.resultChan <- err
					break
				}

				exists := s.sigCache.Exists(types.NewID(sigHash), tx.MintTransaction.Signature, mintKey)
				if exists {
					s.resultChan <- nil
					break
				}

				valid, err := mintKey.Verify(sigHash, tx.MintTransaction.Signature)
				if err != nil {
					s.resultChan <- err
					break
				}
				if !valid {
					s.resultChan <- ruleError(ErrInvalidTx, "stake tx invalid signature")
					break
				}
				s.sigCache.Add(types.NewID(sigHash), tx.MintTransaction.Signature, mintKey)
				s.resultChan <- nil
			case *transactions.Transaction_StakeTransaction:
				validatorID, err := peer.IDFromBytes(tx.StakeTransaction.Validator_ID)
				if err != nil {
					s.resultChan <- ruleError(ErrInvalidTx, "stake tx validator ID does not decode")
					break
				}

				validatorPubkey, err := validatorID.ExtractPublicKey()
				if err != nil {
					s.resultChan <- ruleError(ErrInvalidTx, "stake tx validator pubkey invalid")
					break
				}

				sigHash, err := tx.StakeTransaction.SigHash()
				if err != nil {
					s.resultChan <- err
					break
				}

				exists := s.sigCache.Exists(types.NewID(sigHash), tx.StakeTransaction.Signature, validatorPubkey)
				if exists {
					s.resultChan <- nil
					break
				}

				valid, err := validatorPubkey.Verify(sigHash, tx.StakeTransaction.Signature)
				if err != nil {
					s.resultChan <- err
					break
				}
				if !valid {
					s.resultChan <- ruleError(ErrInvalidTx, "stake tx invalid signature")
					break
				}
				s.sigCache.Add(types.NewID(sigHash), tx.StakeTransaction.Signature, validatorPubkey)
				s.resultChan <- nil
			case *transactions.Transaction_StandardTransaction, *transactions.Transaction_TreasuryTransaction:
				s.resultChan <- nil
			}
		case <-s.done:
			return
		}
	}
}
