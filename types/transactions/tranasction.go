// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package transactions

import (
	"encoding/json"
	"errors"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"google.golang.org/protobuf/proto"
	"time"
)

var _ types.Serializable = (*Transaction)(nil)
var _ types.Serializable = (*StandardTransaction)(nil)
var _ types.Serializable = (*CoinbaseTransaction)(nil)
var _ types.Serializable = (*StakeTransaction)(nil)
var _ types.Serializable = (*TreasuryTransaction)(nil)
var _ types.Serializable = (*MintTransaction)(nil)

// WrapTransaction wraps a typed transaction in the transaction wrapper
func WrapTransaction(tx interface{}) *Transaction {
	var t isTransaction_Tx
	switch typ := tx.(type) {
	case *TreasuryTransaction:
		t = &Transaction_TreasuryTransaction{TreasuryTransaction: typ}
	case *StakeTransaction:
		t = &Transaction_StakeTransaction{StakeTransaction: typ}
	case *StandardTransaction:
		t = &Transaction_StandardTransaction{StandardTransaction: typ}
	case *MintTransaction:
		t = &Transaction_MintTransaction{MintTransaction: typ}
	case *CoinbaseTransaction:
		t = &Transaction_CoinbaseTransaction{CoinbaseTransaction: typ}
	}
	return &Transaction{
		Tx: t,
	}
}

type txWrapperJSON struct {
	StandardTransaction *StandardTransaction `json:"standard_transaction"`
	MintTransaction     *MintTransaction     `json:"mint_transaction"`
	StakeTransaction    *StakeTransaction    `json:"stake_transaction"`
	CoinbaseTransaction *CoinbaseTransaction `json:"coinbase_transaction"`
	TreasuryTransaction *TreasuryTransaction `json:"treasury_transaction"`
}

// ID returns the txid of the transaction. If the id is cached
// it will return from cache instead of recomputing the hash. If
// it is not cached it will cache the computed txid for faster
// returns next time.
func (tx *Transaction) ID() types.ID {
	if len(tx.cachedTxid) > 0 {
		return types.NewID(tx.cachedTxid)
	}
	id := types.NewIDFromData(append(tx.UID().Bytes(), tx.WID().Bytes()...))
	tx.cachedTxid = id.Bytes()
	return id
}

// CacheTxid caches the transaction ID for faster returns
func (tx *Transaction) CacheTxid(txid types.ID) {
	tx.cachedTxid = txid.Bytes()
}

// CacheWid caches the WID for faster returns
func (tx *Transaction) CacheWid(wid types.ID) {
	tx.cachedWid = wid.Bytes()
}

// UID returns the ID of the transaction excluding the proof
func (tx *Transaction) UID() types.ID {
	clone := proto.Clone(tx)
	switch tx := clone.(*Transaction).GetTx().(type) {
	case *Transaction_StandardTransaction:
		tx.StandardTransaction.Proof = nil
	case *Transaction_CoinbaseTransaction:
		tx.CoinbaseTransaction.Proof = nil
	case *Transaction_MintTransaction:
		tx.MintTransaction.Proof = nil
	case *Transaction_TreasuryTransaction:
		tx.TreasuryTransaction.Proof = nil
	case *Transaction_StakeTransaction:
		tx.StakeTransaction.Proof = nil
	}
	ser, _ := clone.(*Transaction).Serialize()
	return types.NewIDFromData(ser)
}

// WID returns the witness ID for the transaction
func (tx *Transaction) WID() types.ID {
	if len(tx.cachedWid) > 0 {
		return types.NewID(tx.cachedWid)
	}
	var proof []byte
	switch t := tx.GetTx().(type) {
	case *Transaction_StandardTransaction:
		proof = t.StandardTransaction.Proof
	case *Transaction_CoinbaseTransaction:
		proof = t.CoinbaseTransaction.Proof
	case *Transaction_MintTransaction:
		proof = t.MintTransaction.Proof
	case *Transaction_TreasuryTransaction:
		proof = t.TreasuryTransaction.Proof
	case *Transaction_StakeTransaction:
		proof = t.StakeTransaction.Proof
	}
	return types.NewIDFromData(proof)
}

// DropProof sets the proof in the transaction to nil
func (tx *Transaction) DropProof() {
	switch t := tx.GetTx().(type) {
	case *Transaction_StandardTransaction:
		t.StandardTransaction.Proof = nil
	case *Transaction_CoinbaseTransaction:
		t.CoinbaseTransaction.Proof = nil
	case *Transaction_MintTransaction:
		t.MintTransaction.Proof = nil
	case *Transaction_TreasuryTransaction:
		t.TreasuryTransaction.Proof = nil
	case *Transaction_StakeTransaction:
		t.StakeTransaction.Proof = nil
	}
}

// Outputs returns the transaction's output
func (tx *Transaction) Outputs() []*Output {
	outputs := make([]*Output, 0, 1)
	switch tx := tx.GetTx().(type) {
	case *Transaction_StandardTransaction:
		outputs = append(outputs, tx.StandardTransaction.Outputs...)
	case *Transaction_CoinbaseTransaction:
		outputs = append(outputs, tx.CoinbaseTransaction.Outputs...)
	case *Transaction_MintTransaction:
		outputs = append(outputs, tx.MintTransaction.Outputs...)
	case *Transaction_TreasuryTransaction:
		outputs = append(outputs, tx.TreasuryTransaction.Outputs...)
	}
	return outputs
}

// Nullifiers returns the transaction's nullifiers
func (tx *Transaction) Nullifiers() []types.Nullifier {
	nullifiers := make([]types.Nullifier, 0, 1)
	switch tx := tx.GetTx().(type) {
	case *Transaction_StandardTransaction:
		for _, n := range tx.StandardTransaction.Nullifiers {
			nullifiers = append(nullifiers, types.NewNullifier(n))
		}
	case *Transaction_MintTransaction:
		for _, n := range tx.MintTransaction.Nullifiers {
			nullifiers = append(nullifiers, types.NewNullifier(n))
		}
	}
	return nullifiers
}

// Type returns the transaction type as a string
func (tx *Transaction) Type() string {
	switch tx.GetTx().(type) {
	case *Transaction_StandardTransaction:
		return "standard"
	case *Transaction_CoinbaseTransaction:
		return "coinbase"
	case *Transaction_MintTransaction:
		return "mint"
	case *Transaction_TreasuryTransaction:
		return "treasury"
	case *Transaction_StakeTransaction:
		return "stake"
	}
	return "unknown"
}

func (tx *Transaction) Serialize() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *Transaction) SerializedSize() (int, error) {
	ser, err := proto.Marshal(tx)
	if err != nil {
		return 0, err
	}
	return len(ser), nil
}

func (tx *Transaction) Deserialize(data []byte) error {
	newTx := &Transaction{}
	if err := proto.Unmarshal(data, newTx); err != nil {
		return err
	}
	tx.Tx = newTx.Tx
	return nil
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	switch tx := tx.GetTx().(type) {
	case *Transaction_StandardTransaction:
		ret := &struct {
			StandardTransaction *StandardTransaction `json:"standard_transaction"`
		}{
			StandardTransaction: tx.StandardTransaction,
		}
		return json.Marshal(ret)
	case *Transaction_CoinbaseTransaction:
		ret := &struct {
			CoinbaseTransaction *CoinbaseTransaction `json:"coinbase_transaction"`
		}{
			CoinbaseTransaction: tx.CoinbaseTransaction,
		}
		return json.Marshal(ret)
	case *Transaction_StakeTransaction:
		ret := &struct {
			StakeTransaction *StakeTransaction `json:"stake_transaction"`
		}{
			StakeTransaction: tx.StakeTransaction,
		}
		return json.Marshal(ret)
	case *Transaction_TreasuryTransaction:
		ret := &struct {
			TreasuryTransaction *TreasuryTransaction `json:"treasury_transaction"`
		}{
			TreasuryTransaction: tx.TreasuryTransaction,
		}
		return json.Marshal(ret)
	case *Transaction_MintTransaction:
		ret := &struct {
			MintTransaction *MintTransaction `json:"mint_transaction"`
		}{
			MintTransaction: tx.MintTransaction,
		}
		return json.Marshal(ret)
	}
	return nil, errors.New("unknown tx type")
}

func (tx *Transaction) UnmarshalJSON(data []byte) error {
	newTx := txWrapperJSON{}
	if err := json.Unmarshal(data, &newTx); err != nil {
		return err
	}
	if newTx.StakeTransaction != nil {
		t := WrapTransaction(newTx.StakeTransaction)
		*tx = *t //nolint:govet
	}
	if newTx.CoinbaseTransaction != nil {
		t := WrapTransaction(newTx.CoinbaseTransaction)
		*tx = *t //nolint:govet
	}
	if newTx.StandardTransaction != nil {
		t := WrapTransaction(newTx.StandardTransaction)
		*tx = *t //nolint:govet
	}
	if newTx.MintTransaction != nil {
		t := WrapTransaction(newTx.MintTransaction)
		*tx = *t //nolint:govet
	}
	if newTx.TreasuryTransaction != nil {
		t := WrapTransaction(newTx.TreasuryTransaction)
		*tx = *t //nolint:govet
	}
	return nil
}

type standardTxJSON struct {
	Outputs    []*Output            `json:"outputs"`
	Nullifiers []types.HexEncodable `json:"nullifiers"`
	TxoRoot    types.HexEncodable   `json:"txo_root"`
	Locktime   *locktimeJSON        `json:"locktime,omitempty"`
	Fee        types.Amount         `json:"fee"`
	Proof      types.HexEncodable   `json:"proof"`
}

type locktimeJSON struct {
	Timestamp int64 `json:"timestamp"`
	Precision int64 `json:"precision"`
}

func (tx *StandardTransaction) Serialize() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *StandardTransaction) Deserialize(data []byte) error {
	newTx := &StandardTransaction{}
	if err := proto.Unmarshal(data, newTx); err != nil {
		return err
	}
	tx.Fee = newTx.Fee
	tx.Locktime = newTx.Locktime
	tx.Proof = newTx.Proof
	tx.Outputs = newTx.Outputs
	tx.TxoRoot = newTx.TxoRoot
	tx.Nullifiers = newTx.Nullifiers
	return nil
}

func (tx *StandardTransaction) SigHash() ([]byte, error) {
	cpy := proto.Clone(tx)
	cpy.(*StandardTransaction).Proof = nil

	b, err := proto.Marshal(cpy)
	if err != nil {
		return nil, err
	}

	return hash.HashFunc(b), nil
}

func (tx *StandardTransaction) ID() types.ID {
	wtx := WrapTransaction(tx)
	return wtx.ID()
}

func (tx *StandardTransaction) ToCircuitParams() (zk.Parameters, error) {
	sigHash, err := tx.SigHash()
	if err != nil {
		return nil, err
	}
	outputs := make([]circparams.PublicOutput, 0, len(tx.Outputs))
	for _, out := range tx.Outputs {
		outputs = append(outputs, circparams.PublicOutput{
			Commitment: types.NewID(out.Commitment),
			CipherText: out.Ciphertext,
		})
	}
	nullifiers := make([]types.Nullifier, 0, len(tx.Nullifiers))
	for _, n := range tx.Nullifiers {
		nullifiers = append(nullifiers, types.NewNullifier(n))
	}
	params := &circparams.StandardPublicParams{
		SigHash:    types.NewID(sigHash),
		Nullifiers: nullifiers,
		TXORoot:    types.NewID(tx.TxoRoot),
		Fee:        types.Amount(tx.Fee),
		Outputs:    outputs,
	}
	if tx.Locktime != nil {
		params.Locktime = time.Unix(tx.Locktime.Timestamp, 0)
		params.LocktimePrecision = time.Duration(tx.Locktime.Precision) * time.Second
	}
	return params, nil
}

func (tx *StandardTransaction) MarshalJSON() ([]byte, error) {
	nullifiers := make([]types.HexEncodable, 0, len(tx.Nullifiers))
	for _, n := range tx.Nullifiers {
		nullifiers = append(nullifiers, n)
	}
	s := &standardTxJSON{
		Outputs:    tx.Outputs,
		Nullifiers: nullifiers,
		TxoRoot:    tx.TxoRoot,
		Fee:        types.Amount(tx.Fee),
		Proof:      tx.Proof,
	}
	if tx.Locktime != nil {
		s.Locktime = &locktimeJSON{
			Timestamp: tx.Locktime.Timestamp,
			Precision: tx.Locktime.Precision,
		}
	}
	return json.Marshal(s)
}

func (tx *StandardTransaction) UnmarshalJSON(data []byte) error {
	newTx := &standardTxJSON{}
	if err := json.Unmarshal(data, newTx); err != nil {
		return err
	}
	nullifiers := make([][]byte, 0, len(newTx.Nullifiers))
	for _, n := range newTx.Nullifiers {
		nullifiers = append(nullifiers, n)
	}
	*tx = StandardTransaction{
		Outputs:    newTx.Outputs,
		Nullifiers: nullifiers,
		TxoRoot:    newTx.TxoRoot,
		Fee:        uint64(newTx.Fee),
		Proof:      newTx.Proof,
	}
	if newTx.Locktime != nil {
		tx.Locktime = &Locktime{
			Timestamp: newTx.Locktime.Timestamp,
			Precision: newTx.Locktime.Precision,
		}
	}
	return nil
}

type coinbaseTxJSON struct {
	Validator_ID types.HexEncodable `json:"validator_ID"`
	NewCoins     types.Amount       `json:"new_coins"`
	Outputs      []*Output          `json:"outputs"`
	Signature    types.HexEncodable `json:"signature"`
	Proof        types.HexEncodable `json:"proof"`
}

func (tx *CoinbaseTransaction) Serialize() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *CoinbaseTransaction) Deserialize(data []byte) error {
	newTx := &CoinbaseTransaction{}
	if err := proto.Unmarshal(data, newTx); err != nil {
		return err
	}
	tx.Signature = newTx.Signature
	tx.Proof = newTx.Proof
	tx.Outputs = newTx.Outputs
	tx.NewCoins = newTx.NewCoins
	tx.Validator_ID = newTx.Validator_ID
	return nil
}

func (tx *CoinbaseTransaction) SigHash() ([]byte, error) {
	cpy := proto.Clone(tx)
	cpy.(*CoinbaseTransaction).Signature = nil
	cpy.(*CoinbaseTransaction).Proof = nil

	b, err := proto.Marshal(cpy)
	if err != nil {
		return nil, err
	}

	return hash.HashFunc(b), nil
}

func (tx *CoinbaseTransaction) ID() types.ID {
	wtx := WrapTransaction(tx)
	return wtx.ID()
}

func (tx *CoinbaseTransaction) ToCircuitParams() (zk.Parameters, error) {
	outputs := make([]circparams.PublicOutput, 0, len(tx.Outputs))
	for _, out := range tx.Outputs {
		outputs = append(outputs, circparams.PublicOutput{
			Commitment: types.NewID(out.Commitment),
			CipherText: out.Ciphertext,
		})
	}
	params := &circparams.CoinbasePublicParams{
		Coinbase: types.Amount(tx.NewCoins),
		Outputs:  outputs,
	}
	return params, nil
}

func (tx *CoinbaseTransaction) MarshalJSON() ([]byte, error) {
	c := &coinbaseTxJSON{
		Validator_ID: tx.Validator_ID,
		NewCoins:     types.Amount(tx.NewCoins),
		Outputs:      tx.Outputs,
		Signature:    tx.Signature,
		Proof:        tx.Proof,
	}
	return json.Marshal(c)
}

func (tx *CoinbaseTransaction) UnmarshalJSON(data []byte) error {
	newTx := &coinbaseTxJSON{}
	if err := json.Unmarshal(data, newTx); err != nil {
		return err
	}
	*tx = CoinbaseTransaction{
		Validator_ID: newTx.Validator_ID,
		NewCoins:     uint64(newTx.NewCoins),
		Outputs:      newTx.Outputs,
		Signature:    newTx.Signature,
		Proof:        newTx.Proof,
	}
	return nil
}

type stakeTxJSON struct {
	Validator_ID types.HexEncodable `json:"validator_ID"`
	Amount       types.Amount       `json:"amount"`
	Nullifier    types.HexEncodable `json:"nullifier"`
	TxoRoot      types.HexEncodable `json:"txo_root"`
	LockedUntil  int64              `json:"locked_until"`
	Signature    types.HexEncodable `json:"signature"`
	Proof        types.HexEncodable `json:"proof"`
}

func (tx *StakeTransaction) Serialize() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *StakeTransaction) Deserialize(data []byte) error {
	newTx := &StakeTransaction{}
	if err := proto.Unmarshal(data, newTx); err != nil {
		return err
	}
	tx.Validator_ID = newTx.Validator_ID
	tx.Proof = newTx.Proof
	tx.TxoRoot = newTx.TxoRoot
	tx.LockedUntil = newTx.LockedUntil
	tx.Signature = newTx.Signature
	tx.Amount = newTx.Amount
	tx.Nullifier = newTx.Nullifier
	return nil
}

func (tx *StakeTransaction) SigHash() ([]byte, error) {
	cpy := proto.Clone(tx)
	cpy.(*StakeTransaction).Signature = nil
	cpy.(*StakeTransaction).Proof = nil

	b, err := proto.Marshal(cpy)
	if err != nil {
		return nil, err
	}

	return hash.HashFunc(b), nil
}

func (tx *StakeTransaction) ID() types.ID {
	wtx := WrapTransaction(tx)
	return wtx.ID()
}

func (tx *StakeTransaction) ToCircuitParams() (zk.Parameters, error) {
	sigHash, err := tx.SigHash()
	if err != nil {
		return nil, err
	}
	params := &circparams.StakePublicParams{
		StakeAmount: types.Amount(tx.Amount),
		SigHash:     types.NewID(sigHash),
		Nullifier:   types.NewNullifier(tx.Nullifier),
		TXORoot:     types.NewID(tx.TxoRoot),
		LockedUntil: time.Unix(tx.LockedUntil, 0),
	}
	return params, nil
}

func (tx *StakeTransaction) MarshalJSON() ([]byte, error) {
	s := &stakeTxJSON{
		Validator_ID: tx.Validator_ID,
		Amount:       types.Amount(tx.Amount),
		Nullifier:    tx.Nullifier,
		TxoRoot:      tx.TxoRoot,
		LockedUntil:  tx.LockedUntil,
		Signature:    tx.Signature,
		Proof:        tx.Proof,
	}
	return json.Marshal(s)
}

func (tx *StakeTransaction) UnmarshalJSON(data []byte) error {
	newTx := &stakeTxJSON{}
	if err := json.Unmarshal(data, newTx); err != nil {
		return err
	}
	*tx = StakeTransaction{
		Validator_ID: newTx.Validator_ID,
		Amount:       uint64(newTx.Amount),
		Nullifier:    newTx.Nullifier,
		TxoRoot:      newTx.TxoRoot,
		LockedUntil:  newTx.LockedUntil,
		Signature:    newTx.Signature,
		Proof:        newTx.Proof,
	}
	return nil
}

type treasuryTxJSON struct {
	Amount       types.Amount       `json:"amount"`
	Outputs      []*Output          `json:"outputs"`
	ProposalHash types.HexEncodable `json:"proposal_hash"`
	Proof        types.HexEncodable `json:"proof"`
}

func (tx *TreasuryTransaction) Serialize() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *TreasuryTransaction) Deserialize(data []byte) error {
	newTx := &TreasuryTransaction{}
	if err := proto.Unmarshal(data, newTx); err != nil {
		return err
	}
	tx.Proof = newTx.Proof
	tx.Outputs = newTx.Outputs
	tx.Outputs = newTx.Outputs
	tx.ProposalHash = newTx.ProposalHash
	return nil
}

func (tx *TreasuryTransaction) SigHash() ([]byte, error) {
	cpy := proto.Clone(tx)
	cpy.(*TreasuryTransaction).Proof = nil

	b, err := proto.Marshal(cpy)
	if err != nil {
		return nil, err
	}

	return hash.HashFunc(b), nil
}

func (tx *TreasuryTransaction) ID() types.ID {
	wtx := WrapTransaction(tx)
	return wtx.ID()
}

func (tx *TreasuryTransaction) ToCircuitParams() (zk.Parameters, error) {
	outputs := make([]circparams.PublicOutput, 0, len(tx.Outputs))
	for _, out := range tx.Outputs {
		outputs = append(outputs, circparams.PublicOutput{
			Commitment: types.NewID(out.Commitment),
			CipherText: out.Ciphertext,
		})
	}
	params := &circparams.TreasuryPublicParams{
		TreasuryWithdrawAmount: types.Amount(tx.Amount),
		Outputs:                outputs,
	}
	return params, nil
}

func (tx *TreasuryTransaction) MarshalJSON() ([]byte, error) {
	t := &treasuryTxJSON{
		Amount:       types.Amount(tx.Amount),
		Outputs:      tx.Outputs,
		ProposalHash: tx.ProposalHash,
		Proof:        tx.Proof,
	}
	return json.Marshal(t)
}

func (tx *TreasuryTransaction) UnmarshalJSON(data []byte) error {
	newTx := &treasuryTxJSON{}
	if err := json.Unmarshal(data, newTx); err != nil {
		return err
	}
	*tx = TreasuryTransaction{
		Amount:       uint64(newTx.Amount),
		Outputs:      newTx.Outputs,
		ProposalHash: newTx.ProposalHash,
		Proof:        newTx.Proof,
	}
	return nil
}

type mintTxJSON struct {
	Type         MintTransaction_AssetType `json:"type"`
	Asset_ID     types.HexEncodable        `json:"asset_ID"`
	DocumentHash types.HexEncodable        `json:"document_hash"`
	NewTokens    uint64                    `json:"new_tokens"`
	Outputs      []*Output                 `json:"outputs"`
	Fee          types.Amount              `json:"fee"`
	Nullifiers   []types.HexEncodable      `json:"nullifiers"`
	TxoRoot      types.HexEncodable        `json:"txo_root"`
	MintKey      types.HexEncodable        `json:"mint_key"`
	Locktime     *locktimeJSON             `json:"locktime,omitempty"`
	Signature    types.HexEncodable        `json:"signature"`
	Proof        types.HexEncodable        `json:"proof"`
}

func (tx *MintTransaction) Serialize() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *MintTransaction) Deserialize(data []byte) error {
	newTx := &MintTransaction{}
	if err := proto.Unmarshal(data, newTx); err != nil {
		return err
	}
	tx.Fee = newTx.Fee
	tx.Locktime = newTx.Locktime
	tx.Proof = newTx.Proof
	tx.Outputs = newTx.Outputs
	tx.TxoRoot = newTx.TxoRoot
	tx.Nullifiers = newTx.Nullifiers
	tx.Asset_ID = newTx.Asset_ID
	tx.Type = newTx.Type
	tx.MintKey = newTx.MintKey
	tx.NewTokens = newTx.NewTokens
	tx.DocumentHash = newTx.DocumentHash
	return nil
}

func (tx *MintTransaction) SigHash() ([]byte, error) {
	cpy := proto.Clone(tx)
	cpy.(*MintTransaction).Signature = nil
	cpy.(*MintTransaction).Proof = nil

	b, err := proto.Marshal(cpy)
	if err != nil {
		return nil, err
	}

	return hash.HashFunc(b), nil
}

func (tx *MintTransaction) ID() types.ID {
	wtx := WrapTransaction(tx)
	return wtx.ID()
}

func (tx *MintTransaction) ToCircuitParams() (zk.Parameters, error) {
	sigHash, err := tx.SigHash()
	if err != nil {
		return nil, err
	}
	outputs := make([]circparams.PublicOutput, 0, len(tx.Outputs))
	for _, out := range tx.Outputs {
		outputs = append(outputs, circparams.PublicOutput{
			Commitment: types.NewID(out.Commitment),
			CipherText: out.Ciphertext,
		})
	}
	nullifiers := make([]types.Nullifier, 0, len(tx.Nullifiers))
	for _, n := range tx.Nullifiers {
		nullifiers = append(nullifiers, types.NewNullifier(n))
	}
	params := &circparams.MintPublicParams{
		SigHash:    types.NewID(sigHash),
		Nullifiers: nullifiers,
		TXORoot:    types.NewID(tx.TxoRoot),
		Fee:        types.Amount(tx.Fee),
		MintID:     types.NewID(tx.Asset_ID),
		MintAmount: types.Amount(tx.NewTokens),
		Outputs:    outputs,
	}
	if tx.Locktime != nil {
		params.Locktime = time.Unix(tx.Locktime.Timestamp, 0)
		params.LocktimePrecision = time.Duration(tx.Locktime.Precision) * time.Second
	}
	return params, nil
}

func (tx *MintTransaction) MarshalJSON() ([]byte, error) {
	nullifiers := make([]types.HexEncodable, 0, len(tx.Nullifiers))
	for _, n := range tx.Nullifiers {
		nullifiers = append(nullifiers, n)
	}
	s := &mintTxJSON{
		Type:         tx.Type,
		Asset_ID:     tx.Asset_ID,
		DocumentHash: tx.DocumentHash,
		NewTokens:    tx.NewTokens,
		Outputs:      tx.Outputs,
		Fee:          types.Amount(tx.Fee),
		Nullifiers:   nullifiers,
		TxoRoot:      tx.TxoRoot,
		MintKey:      tx.MintKey,
		Signature:    tx.Signature,
		Proof:        tx.Proof,
	}
	if tx.Locktime != nil {
		s.Locktime = &locktimeJSON{
			Timestamp: tx.Locktime.Timestamp,
			Precision: tx.Locktime.Precision,
		}
	}
	return json.Marshal(s)
}

func (tx *MintTransaction) UnmarshalJSON(data []byte) error {
	newTx := &mintTxJSON{}
	if err := json.Unmarshal(data, newTx); err != nil {
		return err
	}
	nullifiers := make([][]byte, 0, len(newTx.Nullifiers))
	for _, n := range newTx.Nullifiers {
		nullifiers = append(nullifiers, n)
	}
	*tx = MintTransaction{
		Type:         newTx.Type,
		Asset_ID:     newTx.Asset_ID,
		DocumentHash: newTx.DocumentHash,
		NewTokens:    newTx.NewTokens,
		Outputs:      newTx.Outputs,
		Fee:          uint64(newTx.Fee),
		Nullifiers:   nullifiers,
		TxoRoot:      newTx.TxoRoot,
		MintKey:      newTx.MintKey,
		Signature:    newTx.Signature,
		Proof:        newTx.Proof,
	}
	if newTx.Locktime != nil {
		tx.Locktime = &Locktime{
			Timestamp: newTx.Locktime.Timestamp,
			Precision: newTx.Locktime.Precision,
		}
	}
	return nil
}

type outputJSON struct {
	Commitment types.HexEncodable `json:"commitment"`
	Ciphertext types.HexEncodable `json:"ciphertext"`
}

func (out *Output) MarshalJSON() ([]byte, error) {
	o := &outputJSON{
		Commitment: out.Commitment,
		Ciphertext: out.Ciphertext,
	}
	return json.Marshal(o)
}

func (out *Output) UnmarshalJSON(data []byte) error {
	var o outputJSON
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	*out = Output{
		Commitment: o.Commitment,
		Ciphertext: o.Ciphertext,
	}
	return nil
}
