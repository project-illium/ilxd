// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"bytes"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"google.golang.org/protobuf/proto"
	"time"
)

const minTimeAfterRejection = time.Minute

type processTxReq struct {
	tx         *transactions.Transaction
	resultChan chan error
}
type validateTxReq struct {
	tx         *transactions.Transaction
	resultChan chan error
}
type removeBlockTxsReq struct {
	txs []*transactions.Transaction
}
type getTransactionReq struct {
	txid       types.ID
	resultChan chan *transactions.Transaction
}
type getTransactionsReq struct {
	resultChan chan map[types.ID]*transactions.Transaction
}
type rejectedBlockReq struct {
	blk *blocks.Block
}
type poolTx struct {
	tx            *transactions.Transaction
	firstSeen     time.Time
	rejectedBlock bool
}

// Mempool holds valid transactions that have been relayed around the
// network but have not yet made it into a block. The pool will validate
// transactions before admitting them. The block generation package uses
// the mempool transactions to generate blocks.
type Mempool struct {
	pool           map[types.ID]*poolTx
	nullifiers     map[types.Nullifier]types.ID
	treasuryDebits map[types.ID]types.Amount
	coinbases      map[peer.ID]*transactions.CoinbaseTransaction
	cfg            *config
	msgChan        chan interface{}
	quit           chan struct{}
}

// NewMempool returns a new mempool with the configuration options.
//
// The options include local node policy for determining which (valid)
// transactions to admit into the mempool.
func NewMempool(opts ...Option) (*Mempool, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	m := &Mempool{
		pool:           make(map[types.ID]*poolTx),
		nullifiers:     make(map[types.Nullifier]types.ID),
		treasuryDebits: make(map[types.ID]types.Amount),
		coinbases:      make(map[peer.ID]*transactions.CoinbaseTransaction),
		cfg:            &cfg,
		msgChan:        make(chan interface{}),
		quit:           make(chan struct{}),
	}
	go m.validationHandler()
	return m, nil
}

// Close shuts down the validationHandlder and stops the mempool.
func (m *Mempool) Close() {
	close(m.quit)
}

func (m *Mempool) validationHandler() {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case msg := <-m.msgChan:
			switch req := msg.(type) {
			case *processTxReq:
				req.resultChan <- m.processTransaction(req.tx)
			case *removeBlockTxsReq:
				m.removeBlockTransactions(req.txs)
			case *getTransactionReq:
				req.resultChan <- m.handleGetTransaction(req.txid)
			case *getTransactionsReq:
				req.resultChan <- m.handleGetTransactions()
			case *validateTxReq:
				req.resultChan <- m.handleValidateTransaction(req.tx)
			case *rejectedBlockReq:
				m.handleRejectedBlock(req.blk)
			}
		case <-ticker.C:
			toDelete := make([]*transactions.Transaction, 0)
			for _, tx := range m.pool {
				if time.Now().After(tx.firstSeen.Add(m.cfg.transactionTTL)) {
					toDelete = append(toDelete, tx.tx)
				}
			}
			if len(toDelete) > 0 {
				m.removeBlockTransactions(toDelete)
			}
		case <-m.quit:
			return
		}
	}
}

// ProcessTransaction evaluates a transaction and accepts it into the mempool if
// it passes all validation checks.
//
// This method is safe for concurrent access. It will attempt to do all the static
// validation, such as the expensive signature and proof checks, without locking.
// The rest of validation, such as nullifier checks, duplicate mempool checks, etc.
// are done in a single threaded channel.
func (m *Mempool) ProcessTransaction(tx *transactions.Transaction) error {
	if err := blockchain.CheckTransactionSanity(tx, time.Now()); err != nil {
		return err
	}

	fpkb, isFeePayer, err := policy.CalcFeePerKilobyte(tx)
	if err != nil {
		return err
	}
	if isFeePayer && fpkb < m.cfg.policy.GetMinFeePerKilobyte() {
		return policyError(ErrFeeTooLow, "transaction fee is below policy minimum")
	}

	proofChan := blockchain.ValidateTransactionProof(tx, m.cfg.proofCache, m.cfg.verifier)
	sigChan := blockchain.ValidateTransactionSig(tx, m.cfg.sigCache)

	err = <-proofChan
	if err != nil {
		return err
	}
	err = <-sigChan
	if err != nil {
		return err
	}

	resultChan := make(chan error)
	defer close(resultChan)
	m.msgChan <- &processTxReq{
		tx:         proto.Clone(tx).(*transactions.Transaction),
		resultChan: resultChan,
	}
	err = <-resultChan
	return err
}

// GetTransaction returns a transaction given the ID if it exists in the pool.
//
// This method is safe for concurrent access.
func (m *Mempool) GetTransaction(txid types.ID) (*transactions.Transaction, error) {
	resp := make(chan *transactions.Transaction)
	defer close(resp)
	m.msgChan <- &getTransactionReq{
		txid:       txid,
		resultChan: resp,
	}
	tx := <-resp

	if tx == nil {
		return nil, ErrNotFound
	}
	cpy := proto.Clone(tx)
	return cpy.(*transactions.Transaction), nil
}

// handleGetTransaction is the implementation for GetTransaction
//
// This method is NOT safe for concurrent access.
func (m *Mempool) handleGetTransaction(txid types.ID) *transactions.Transaction {
	ttlTx, ok := m.pool[txid]
	if !ok {
		return nil
	}
	return ttlTx.tx
}

// GetTransactions returns the full list of transactions from the pool.
//
// This method is safe for concurrent access.
func (m *Mempool) GetTransactions() map[types.ID]*transactions.Transaction {
	resp := make(chan map[types.ID]*transactions.Transaction)
	defer close(resp)
	m.msgChan <- &getTransactionsReq{
		resultChan: resp,
	}
	pool := <-resp

	return pool
}

// handleGetTransactions is the implementation for GetTransactions
//
// This method is NOT safe for concurrent access.
func (m *Mempool) handleGetTransactions() map[types.ID]*transactions.Transaction {
	pool := make(map[types.ID]*transactions.Transaction)
	for id, tx := range m.pool {
		cpy := proto.Clone(tx.tx)
		pool[id] = cpy.(*transactions.Transaction)
	}
	return pool
}

// RemoveBlockTransactions should be called when a block is connected. It will remove
// the block's transactions from the mempool and update the rest of the mempool state.
//
// This method is safe for concurrent access.
func (m *Mempool) RemoveBlockTransactions(txs []*transactions.Transaction) {
	m.msgChan <- &removeBlockTxsReq{
		txs: txs,
	}
}

// removeBlockTransactions is the implementation for RemoveBlockTransactions.
//
// This method is NOT safe for concurrent access.
func (m *Mempool) removeBlockTransactions(txs []*transactions.Transaction) {
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
				poolID, ok := m.nullifiers[types.NewNullifier(n)]
				if ok {
					delete(m.nullifiers, types.NewNullifier(n))
					delete(m.pool, poolID)
				}
			}
		case *transactions.Transaction_MintTransaction:
			for _, n := range t.MintTransaction.Nullifiers {
				poolID, ok := m.nullifiers[types.NewNullifier(n)]
				if ok {
					delete(m.nullifiers, types.NewNullifier(n))
					delete(m.pool, poolID)
				}
			}
		case *transactions.Transaction_TreasuryTransaction:
			delete(m.treasuryDebits, t.TreasuryTransaction.ID())
		}
	}
}

// ProcessRejectedBlock should be called when a block is rejected. The
// mempool will take a look at the transactions and decide if any should
// be removed from the pool.
//
// This method is safe for concurrent access.
func (m *Mempool) ProcessRejectedBlock(blk *blocks.Block) {
	m.msgChan <- &rejectedBlockReq{
		blk: blk,
	}
}

// handleRejectedBlock is the implementation for ProcessRejectedBlock.
//
// This method is NOT safe for concurrent access.
func (m *Mempool) handleRejectedBlock(blk *blocks.Block) {
	for _, tx := range blk.Transactions {
		ptx, ok := m.pool[tx.ID()]
		if ok {
			// If this transaction was previously in a block that also got rejected, and
			// it's been in the pool for a little while without being included in any other
			// block, let's delete it from the pool.
			if ptx.rejectedBlock && time.Now().After(ptx.firstSeen.Add(minTimeAfterRejection)) {
				delete(m.pool, tx.ID())
				continue
			}

			// Otherwise mark it as having been part of a rejected block.
			ptx.rejectedBlock = true
		}
	}
}

// ValidateTransaction returns whether the transaction is valid and can be added to the
// pool but does *not* add it to the pool.
//
// This function is safe for concurrent access.
func (m *Mempool) ValidateTransaction(tx *transactions.Transaction, validatedProof bool) error {
	if err := blockchain.CheckTransactionSanity(tx, time.Now()); err != nil {
		return err
	}

	fpkb, isFeePayer, err := policy.CalcFeePerKilobyte(tx)
	if err != nil {
		return err
	}
	if isFeePayer && fpkb < m.cfg.policy.GetMinFeePerKilobyte() {
		return policyError(ErrFeeTooLow, "transaction fee is below policy minimum")
	}

	if validatedProof {
		proofChan := blockchain.ValidateTransactionProof(tx, m.cfg.proofCache, m.cfg.verifier)
		sigChan := blockchain.ValidateTransactionSig(tx, m.cfg.sigCache)

		err = <-proofChan
		if err != nil {
			return err
		}
		err = <-sigChan
		if err != nil {
			return err
		}
	}

	respCh := make(chan error)
	defer close(respCh)
	m.msgChan <- &validateTxReq{
		tx:         tx,
		resultChan: respCh,
	}
	err = <-respCh
	return err
}

// handleValidateTransaction is the implementation for ValidateTransaction.
//
// This method is NOT safe for concurrent access.
func (m *Mempool) handleValidateTransaction(tx *transactions.Transaction) error {
	switch t := tx.GetTx().(type) {
	case *transactions.Transaction_CoinbaseTransaction:
		validatorID, err := peer.IDFromBytes(t.CoinbaseTransaction.Validator_ID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "coinbase tx validator ID does not decode")
		}
		validator, err := m.cfg.chainView.GetValidator(validatorID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "validator does not exist in validator set")
		}
		if types.Amount(t.CoinbaseTransaction.NewCoins) != validator.UnclaimedCoins || t.CoinbaseTransaction.NewCoins == 0 {
			return ruleError(blockchain.ErrInvalidTx, "coinbase transaction creates invalid number of coins")
		}

		if !m.cfg.policy.GetValidatorAcceptableCoinbase(validatorID) {
			return policyError(ErrPoorValidatorUptime, "coinbase for peer with poor uptime")
		}

		if !bytes.Equal(m.cfg.chainView.GetEpoch().Bytes(), t.CoinbaseTransaction.Epoch) {
			return ruleError(blockchain.ErrInvalidTx, "coinbase transaction invalid epoch")
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
	case *transactions.Transaction_StakeTransaction:
		if types.Amount(t.StakeTransaction.Amount) < m.cfg.policy.GetMinStake() {
			return policyError(ErrMinStake, "stake amount below policy minimum")
		}
		if _, ok := m.nullifiers[types.NewNullifier(t.StakeTransaction.Nullifier)]; ok {
			return ruleError(blockchain.ErrDoubleSpend, "stake nullifier already in mempool")
		}
		valID, err := peer.IDFromBytes(t.StakeTransaction.Validator_ID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "stake tx validator ID does not decode")
		}
		validator, err := m.cfg.chainView.GetValidator(valID)
		if err == nil {
			stake, exists := validator.Nullifiers[types.NewNullifier(t.StakeTransaction.Nullifier)]
			if exists {
				if stake.Blockstamp.Add(blockchain.ValidatorExpiration - blockchain.RestakePeriod).After(time.Now()) {
					return ruleError(blockchain.ErrRestakeTooEarly, "restake transaction too early")
				}
			}
		}
		exists, err := m.cfg.chainView.NullifierExists(types.NewNullifier(t.StakeTransaction.Nullifier))
		if err != nil {
			return err
		}
		if exists {
			return ruleError(blockchain.ErrDoubleSpend, "tx contains spent nullifier")
		}
		exists, err = m.cfg.chainView.TxoRootExists(types.NewID(t.StakeTransaction.TxoRoot))
		if err != nil {
			return err
		}
		if !exists {
			return ruleError(blockchain.ErrInvalidTx, "txo root does not exist in chain")
		}
	case *transactions.Transaction_TreasuryTransaction:
		whitelist := m.cfg.policy.GetTreasuryWhitelist()
		exists := false
		for _, txid := range whitelist {
			if txid.Compare(tx.ID()) == 0 {
				exists = true
				break
			}
		}
		if !exists {
			return policyError(ErrTreasuryWhitelist, "treasury transaction not whitelisted")
		}

		treasuryBalance, err := m.cfg.chainView.TreasuryBalance()
		if err != nil {
			return err
		}
		inPoolBalance := types.Amount(0)
		for _, amt := range m.treasuryDebits {
			inPoolBalance += amt
		}

		if types.Amount(t.TreasuryTransaction.Amount) > treasuryBalance-inPoolBalance {
			return ruleError(blockchain.ErrInvalidTx, "treasury tx amount exceeds treasury balance")
		}

	default:
		return ruleError(blockchain.ErrInvalidTx, "unknown transaction type")
	}
	return nil
}

// processTransaction is the implementation for ProcessTransaction.
//
// This method is NOT safe for concurrent access.
func (m *Mempool) processTransaction(tx *transactions.Transaction) error {
	if _, ok := m.pool[tx.ID()]; ok {
		return ErrDuplicateTx
	}

	switch t := tx.GetTx().(type) {
	case *transactions.Transaction_CoinbaseTransaction:
		validatorID, err := peer.IDFromBytes(t.CoinbaseTransaction.Validator_ID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "coinbase tx validator ID does not decode")
		}
		validator, err := m.cfg.chainView.GetValidator(validatorID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "validator does not exist in validator set")
		}
		if types.Amount(t.CoinbaseTransaction.NewCoins) != validator.UnclaimedCoins || t.CoinbaseTransaction.NewCoins == 0 {
			return ruleError(blockchain.ErrInvalidTx, "coinbase transaction creates invalid number of coins")
		}

		if !m.cfg.policy.GetValidatorAcceptableCoinbase(validatorID) {
			return policyError(ErrPoorValidatorUptime, "coinbase for peer with poor uptime")
		}

		if !bytes.Equal(m.cfg.chainView.GetEpoch().Bytes(), t.CoinbaseTransaction.Epoch) {
			return ruleError(blockchain.ErrInvalidTx, "coinbase transaction invalid epoch")
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
				return ruleError(ErrDuplicateCoinbase, "coinbase from validator already in pool")
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
		if types.Amount(t.StakeTransaction.Amount) < m.cfg.policy.GetMinStake() {
			return policyError(ErrMinStake, "stake amount below policy minimum")
		}
		if _, ok := m.nullifiers[types.NewNullifier(t.StakeTransaction.Nullifier)]; ok {
			return ruleError(blockchain.ErrDoubleSpend, "stake nullifier already in mempool")
		}
		valID, err := peer.IDFromBytes(t.StakeTransaction.Validator_ID)
		if err != nil {
			return ruleError(blockchain.ErrInvalidTx, "stake tx validator ID does not decode")
		}
		validator, err := m.cfg.chainView.GetValidator(valID)
		if err == nil {
			stake, exists := validator.Nullifiers[types.NewNullifier(t.StakeTransaction.Nullifier)]
			if exists {
				if stake.Blockstamp.Add(blockchain.ValidatorExpiration - blockchain.RestakePeriod).After(time.Now()) {
					return ruleError(blockchain.ErrRestakeTooEarly, "restake transaction too early")
				}
			}
		}
		exists, err := m.cfg.chainView.NullifierExists(types.NewNullifier(t.StakeTransaction.Nullifier))
		if err != nil {
			return err
		}
		if exists {
			return ruleError(blockchain.ErrDoubleSpend, "tx contains spent nullifier")
		}
		exists, err = m.cfg.chainView.TxoRootExists(types.NewID(t.StakeTransaction.TxoRoot))
		if err != nil {
			return err
		}
		if !exists {
			return ruleError(blockchain.ErrInvalidTx, "txo root does not exist in chain")
		}
	case *transactions.Transaction_TreasuryTransaction:
		whitelist := m.cfg.policy.GetTreasuryWhitelist()
		exists := false
		for _, txid := range whitelist {
			if txid.Compare(tx.ID()) == 0 {
				exists = true
				break
			}
		}
		if !exists {
			return policyError(ErrTreasuryWhitelist, "treasury transaction not whitelisted")
		}

		treasuryBalance, err := m.cfg.chainView.TreasuryBalance()
		if err != nil {
			return err
		}
		inPoolBalance := types.Amount(0)
		for _, amt := range m.treasuryDebits {
			inPoolBalance += amt
		}

		if types.Amount(t.TreasuryTransaction.Amount) > treasuryBalance-inPoolBalance {
			return ruleError(blockchain.ErrInvalidTx, "treasury tx amount exceeds treasury balance")
		}

		m.treasuryDebits[t.TreasuryTransaction.ID()] = types.Amount(t.TreasuryTransaction.Amount)
	default:
		return ruleError(blockchain.ErrInvalidTx, "unknown transaction type")
	}

	m.pool[tx.ID()] = &poolTx{
		tx:        tx,
		firstSeen: time.Now(),
	}
	log.Debug("New mempool transaction", log.ArgsFromMap(map[string]any{
		"txid": tx.ID().String(),
		"type": tx.Type(),
	}))
	return nil
}
