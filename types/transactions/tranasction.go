// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package transactions

import (
	"encoding/json"
	"errors"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"google.golang.org/protobuf/proto"
)

var _ types.Serializable = (*Transaction)(nil)
var _ types.Serializable = (*StandardTransaction)(nil)
var _ types.Serializable = (*CoinbaseTransaction)(nil)
var _ types.Serializable = (*StakeTransaction)(nil)
var _ types.Serializable = (*TreasuryTransaction)(nil)
var _ types.Serializable = (*MintTransaction)(nil)

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

func (tx *Transaction) ID() types.ID {
	ser, _ := tx.Serialize()
	return types.NewIDFromData(ser)
}

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
		*tx = *t
	}
	if newTx.CoinbaseTransaction != nil {
		t := WrapTransaction(newTx.CoinbaseTransaction)
		*tx = *t
	}
	if newTx.StandardTransaction != nil {
		t := WrapTransaction(newTx.StandardTransaction)
		*tx = *t
	}
	if newTx.MintTransaction != nil {
		t := WrapTransaction(newTx.MintTransaction)
		*tx = *t
	}
	if newTx.TreasuryTransaction != nil {
		t := WrapTransaction(newTx.TreasuryTransaction)
		*tx = *t
	}
	return nil
}

type standardTxJSON struct {
	Outputs    []*Output            `json:"outputs"`
	Nullifiers []types.HexEncodable `json:"nullifiers"`
	TxoRoot    types.HexEncodable   `json:"txo_root"`
	Locktime   int64                `json:"locktime"`
	Fee        uint64               `json:"fee"`
	Proof      types.HexEncodable   `json:"proof"`
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

func (tx *StandardTransaction) MarshalJSON() ([]byte, error) {
	nullifiers := make([]types.HexEncodable, 0, len(tx.Nullifiers))
	for _, n := range tx.Nullifiers {
		nullifiers = append(nullifiers, n)
	}
	s := &standardTxJSON{
		Outputs:    tx.Outputs,
		Nullifiers: nullifiers,
		TxoRoot:    tx.TxoRoot,
		Locktime:   tx.Locktime,
		Fee:        tx.Fee,
		Proof:      tx.Proof,
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
		Locktime:   newTx.Locktime,
		Fee:        newTx.Fee,
		Proof:      newTx.Proof,
	}
	return nil
}

type coinbaseTxJSON struct {
	Validator_ID types.HexEncodable `json:"validator_ID"`
	NewCoins     uint64             `json:"new_coins"`
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

func (tx *CoinbaseTransaction) MarshalJSON() ([]byte, error) {
	c := &coinbaseTxJSON{
		Validator_ID: tx.Validator_ID,
		NewCoins:     tx.NewCoins,
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
		NewCoins:     newTx.NewCoins,
		Outputs:      newTx.Outputs,
		Signature:    newTx.Signature,
		Proof:        newTx.Proof,
	}
	return nil
}

type stakeTxJSON struct {
	Validator_ID types.HexEncodable `json:"validator_ID"`
	Amount       uint64             `json:"amount"`
	Nullifier    types.HexEncodable `json:"nullifier"`
	TxoRoot      types.HexEncodable `json:"txo_root"`
	Locktime     int64              `json:"locktime"`
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
	tx.Locktime = newTx.Locktime
	tx.Proof = newTx.Proof
	tx.TxoRoot = newTx.TxoRoot
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

func (tx *StakeTransaction) MarshalJSON() ([]byte, error) {
	s := &stakeTxJSON{
		Validator_ID: tx.Validator_ID,
		Amount:       tx.Amount,
		Nullifier:    tx.Nullifier,
		TxoRoot:      tx.TxoRoot,
		Locktime:     tx.Locktime,
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
		Amount:       newTx.Amount,
		Nullifier:    newTx.Nullifier,
		TxoRoot:      newTx.TxoRoot,
		Locktime:     newTx.Locktime,
		Signature:    newTx.Signature,
		Proof:        newTx.Proof,
	}
	return nil
}

type treasuryTxJSON struct {
	Amount       uint64             `json:"amount"`
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

func (tx *TreasuryTransaction) MarshalJSON() ([]byte, error) {
	t := &treasuryTxJSON{
		Amount:       tx.Amount,
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
		Amount:       newTx.Amount,
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
	Fee          uint64                    `json:"fee"`
	Nullifiers   []types.HexEncodable      `json:"nullifiers"`
	TxoRoot      types.HexEncodable        `json:"txo_root"`
	MintKey      types.HexEncodable        `json:"mint_key"`
	Locktime     int64                     `json:"locktime"`
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
		Fee:          tx.Fee,
		Nullifiers:   nullifiers,
		TxoRoot:      tx.TxoRoot,
		MintKey:      tx.MintKey,
		Locktime:     tx.Locktime,
		Signature:    tx.Signature,
		Proof:        tx.Proof,
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
		Fee:          newTx.Fee,
		Nullifiers:   nullifiers,
		TxoRoot:      newTx.TxoRoot,
		MintKey:      newTx.MintKey,
		Locktime:     newTx.Locktime,
		Signature:    newTx.Signature,
		Proof:        newTx.Proof,
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
