// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"sync"
	"time"
)

type Mempool struct {
	pool           map[types.ID]*transactions.Transaction
	nullifiers     map[types.Nullifier]types.ID
	treasuryDebits map[types.ID]uint64
	coinbases      map[peer.ID]*transactions.CoinbaseTransaction
	cfg            *config
	mempoolLock    sync.RWMutex
}

func NewMempool(opts ...Option) (*Mempool, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	m := &Mempool{
		pool:           make(map[types.ID]*transactions.Transaction),
		nullifiers:     make(map[types.Nullifier]types.ID),
		treasuryDebits: make(map[types.ID]uint64),
		coinbases:      make(map[peer.ID]*transactions.CoinbaseTransaction),
		cfg:            &cfg,
		mempoolLock:    sync.RWMutex{},
	}
	return m, nil
}

func (m *Mempool) ProcessTransaction(tx *transactions.Transaction) error {
	m.mempoolLock.Lock()
	defer m.mempoolLock.Unlock()

	if _, ok := m.pool[tx.ID()]; ok {
		return ErrDuplicateTx
	}

	if err := blockchain.CheckTransactionSanity(tx, time.Now()); err != nil {
		return err
	}

	fpb, isFeePayer, err := calcFeePerByte(tx)
	if err != nil {
		return err
	}
	if isFeePayer && fpb < m.cfg.fpb {
		return policyError(ErrFeeTooLow, "transaction fee is below policy minimum")
	}

	switch t := tx.GetTx().(type) {
	case *transactions.Transaction_CoinbaseTransaction:
		validatorID, err := peer.IDFromBytes(t.CoinbaseTransaction.Validator_ID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "coinbase tx validator ID does not decode")
		}
		unclaimedCoins, err := m.cfg.chainView.UnclaimedCoins(validatorID)
		if err != nil {
			return err
		}
		if t.CoinbaseTransaction.NewCoins != unclaimedCoins {
			return ruleError(blockchain.ErrInvalidTx, "coinbase transaction creates invalid number of coins")
		}

		// There is an unlikely scenario where a coinbase could sit in the mempool
		// for an entire epoch and not get included in a block. We don't want two
		// coinbases from the same validator in the mempool so we will evict the
		// older one *only* if the new coinbase is redeeming more unclaimed coins.
		prevCoinbase, ok := m.coinbases[validatorID]
		if ok {
			if t.CoinbaseTransaction.NewCoins > prevCoinbase.NewCoins {
				delete(m.pool, prevCoinbase.ID())
				m.coinbases[validatorID] = t.CoinbaseTransaction
			} else {
				return policyError(ErrDuplicateCoinbase, "coinbase from validator already in pool")
			}
		} else {
			m.coinbases[validatorID] = t.CoinbaseTransaction
		}

	case *transactions.Transaction_StandardTransaction:
		for _, n := range t.StandardTransaction.Nullifiers {
			if _, ok := m.nullifiers[types.NewNullifier(n)]; ok {
				return ruleError(blockchain.ErrDoubleSpend, "nullifier already in mempool")
			}
			exists, err := m.cfg.chainView.NullifierExists(types.NewNullifier(n))
			if err != nil {
				return err
			}
			if exists {
				return ruleError(blockchain.ErrDoubleSpend, "tx contains spent nullifier")
			}
		}
		exists, err := m.cfg.chainView.TxoRootExists(types.NewID(t.StandardTransaction.TxoRoot))
		if err != nil {
			return err
		}
		if !exists {
			return ruleError(blockchain.ErrInvalidTx, "txo root does not exist in chain")
		}
		for _, n := range t.StandardTransaction.Nullifiers {
			m.nullifiers[types.NewNullifier(n)] = t.StandardTransaction.ID()
		}
	case *transactions.Transaction_MintTransaction:
		for _, n := range t.MintTransaction.Nullifiers {
			if _, ok := m.nullifiers[types.NewNullifier(n)]; ok {
				return ruleError(blockchain.ErrDoubleSpend, "nullifier already in mempool")
			}
			exists, err := m.cfg.chainView.NullifierExists(types.NewNullifier(n))
			if err != nil {
				return err
			}
			if exists {
				return ruleError(blockchain.ErrDoubleSpend, "tx contains spent nullifier")
			}
		}
		exists, err := m.cfg.chainView.TxoRootExists(types.NewID(t.MintTransaction.TxoRoot))
		if err != nil {
			return err
		}
		if !exists {
			return ruleError(blockchain.ErrInvalidTx, "txo root does not exist in chain")
		}
		for _, n := range t.MintTransaction.Nullifiers {
			m.nullifiers[types.NewNullifier(n)] = t.MintTransaction.ID()
		}
	case *transactions.Transaction_StakeTransaction:
		if t.StakeTransaction.Amount < m.cfg.minStake {
			return policyError(ErrMinStake, "stake amount below policy minimum")
		}
		if _, ok := m.nullifiers[types.NewNullifier(t.StakeTransaction.Nullifier)]; ok {
			return ruleError(blockchain.ErrDoubleSpend, "stake nullifier already in mempool")
		}
		exists, err := m.cfg.chainView.TxoRootExists(types.NewID(t.StakeTransaction.TxoRoot))
		if err != nil {
			return err
		}
		if !exists {
			return ruleError(blockchain.ErrInvalidTx, "txo root does not exist in chain")
		}
		exists, err = m.cfg.chainView.NullifierExists(types.NewNullifier(t.StakeTransaction.Nullifier))
		if err != nil {
			return err
		}
		if exists {
			return ruleError(blockchain.ErrDoubleSpend, "tx contains spent nullifier")
		}
	case *transactions.Transaction_TreasuryTransaction:
		treasuryBalance, err := m.cfg.chainView.TreasuryBalance()
		if err != nil {
			return err
		}
		inPoolBalance := uint64(0)
		for _, amt := range m.treasuryDebits {
			inPoolBalance += amt
		}

		if inPoolBalance+t.TreasuryTransaction.Amount > treasuryBalance {
			return ruleError(blockchain.ErrInvalidTx, "treasury tx amount exceeds treasury balance")
		}

		m.treasuryDebits[t.TreasuryTransaction.ID()] = t.TreasuryTransaction.Amount
	}

	if err := blockchain.ValidateTransactionSig(tx, m.cfg.sigCache); err != nil {
		return err
	}
	if err := blockchain.ValidateTransactionProof(tx, m.cfg.proofCache); err != nil {
		return err
	}
	m.pool[tx.ID()] = tx
	return nil
}

func (m *Mempool) RemoveTransactions(txs []*transactions.Transaction) {
	m.mempoolLock.Lock()
	defer m.mempoolLock.Unlock()

	for _, tx := range txs {
		delete(m.pool, tx.ID())

		switch t := tx.GetTx().(type) {
		case *transactions.Transaction_CoinbaseTransaction:
			validatorID, err := peer.IDFromBytes(t.CoinbaseTransaction.Validator_ID)
			if err != nil {
				continue
			}
			prevCoinbase, ok := m.coinbases[validatorID]
			if ok {
				if prevCoinbase.ID() == t.CoinbaseTransaction.ID() {
					delete(m.coinbases, validatorID)
				}
			}
		case *transactions.Transaction_StandardTransaction:
			for _, n := range t.StandardTransaction.Nullifiers {
				delete(m.nullifiers, types.NewNullifier(n))
			}
		case *transactions.Transaction_MintTransaction:
			for _, n := range t.MintTransaction.Nullifiers {
				delete(m.nullifiers, types.NewNullifier(n))
			}
		case *transactions.Transaction_TreasuryTransaction:
			delete(m.treasuryDebits, t.TreasuryTransaction.ID())
		}
	}
}

func calcFeePerByte(tx *transactions.Transaction) (uint64, bool, error) {
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

	return uint64(float64(fee) / float64(size)), true, nil
}
