// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package pb

import (
	"encoding/json"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
)

type rawTxJSON struct {
	Tx      *transactions.Transaction `json:"tx"`
	Inputs  []*PrivateInput           `json:"privateInputs"`
	Outputs []*PrivateOutput          `json:"privateOutputs"`
}

func (r *RawTransaction) MarshalJSON() ([]byte, error) {
	s := &rawTxJSON{
		Tx:      r.Tx,
		Inputs:  r.Inputs,
		Outputs: r.Outputs,
	}
	return json.Marshal(s)
}

func (r *RawTransaction) UnmarshalJSON(data []byte) error {
	newTx := &rawTxJSON{}
	if err := json.Unmarshal(data, newTx); err != nil {
		return err
	}

	*r = RawTransaction{
		Tx:      newTx.Tx,
		Inputs:  newTx.Inputs,
		Outputs: newTx.Outputs,
	}
	return nil
}

type privateInputJSON struct {
	Amount           uint64               `json:"amount"`
	AssetID          types.HexEncodable   `json:"assetID"`
	Salt             types.HexEncodable   `json:"salt"`
	State            types.HexEncodable   `json:"state"`
	TxoProof         *TxoProof            `json:"txoProof"`
	ScriptCommitment types.HexEncodable   `json:"scriptCommitment"`
	LockingParams    []types.HexEncodable `json:"lockingParams"`
	UnlockingParams  string               `json:"unlockingParams"`
}

func (i *PrivateInput) MarshalJSON() ([]byte, error) {
	params := make([]types.HexEncodable, 0, len(i.LockingParams))
	for _, p := range i.LockingParams {
		params = append(params, p)
	}
	s := &privateInputJSON{
		Amount:           i.Amount,
		Salt:             i.Salt,
		AssetID:          i.Asset_ID,
		State:            i.State,
		ScriptCommitment: i.ScriptCommitment,
		LockingParams:    params,
		TxoProof:         i.TxoProof,
		UnlockingParams:  i.UnlockingParams,
	}
	return json.Marshal(s)
}

func (i *PrivateInput) UnmarshalJSON(data []byte) error {
	input := &privateInputJSON{}
	if err := json.Unmarshal(data, input); err != nil {
		return err
	}

	params := make([][]byte, 0, len(input.LockingParams))
	for _, p := range input.LockingParams {
		params = append(params, p)
	}

	*i = PrivateInput{
		Amount:           input.Amount,
		Salt:             input.Salt,
		Asset_ID:         input.AssetID,
		State:            input.State,
		ScriptCommitment: input.ScriptCommitment,
		LockingParams:    params,
		TxoProof:         input.TxoProof,
		UnlockingParams:  input.UnlockingParams,
	}
	return nil
}

type privateOutputJSON struct {
	ScriptHash types.HexEncodable `json:"scriptHash"`
	Amount     uint64             `json:"amount"`
	AssetID    types.HexEncodable `json:"assetID"`
	Salt       types.HexEncodable `json:"salt"`
	State      types.HexEncodable `json:"state"`
}

func (o *PrivateOutput) MarshalJSON() ([]byte, error) {
	s := &privateOutputJSON{
		Amount:     o.Amount,
		Salt:       o.Salt,
		AssetID:    o.Asset_ID,
		State:      o.State,
		ScriptHash: o.ScriptHash,
	}
	return json.Marshal(s)
}

func (o *PrivateOutput) UnmarshalJSON(data []byte) error {
	input := &privateOutputJSON{}
	if err := json.Unmarshal(data, input); err != nil {
		return err
	}

	*o = PrivateOutput{
		Amount:     input.Amount,
		Salt:       input.Salt,
		Asset_ID:   input.AssetID,
		State:      input.State,
		ScriptHash: input.ScriptHash,
	}
	return nil
}

type txoProofJSON struct {
	Commitment types.HexEncodable   `json:"commitment"`
	Hashes     []types.HexEncodable `json:"hashes"`
	Flags      uint64               `json:"flags"`
	Index      uint64               `json:"index"`
}

func (t *TxoProof) MarshalJSON() ([]byte, error) {
	hashes := make([]types.HexEncodable, 0, len(t.Hashes))
	for _, h := range t.Hashes {
		hashes = append(hashes, h)
	}
	s := &txoProofJSON{
		Commitment: t.Commitment,
		Hashes:     hashes,
		Flags:      t.Flags,
		Index:      t.Index,
	}
	return json.Marshal(s)
}

func (t *TxoProof) UnmarshalJSON(data []byte) error {
	input := &txoProofJSON{}
	if err := json.Unmarshal(data, input); err != nil {
		return err
	}

	hashes := make([][]byte, 0, len(input.Hashes))
	for _, h := range input.Hashes {
		hashes = append(hashes, h)
	}

	*t = TxoProof{
		Commitment: input.Commitment,
		Hashes:     hashes,
		Flags:      input.Flags,
		Index:      input.Index,
	}
	return nil
}
