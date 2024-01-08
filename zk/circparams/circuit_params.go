// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package circparams

import (
	"fmt"
	"github.com/project-illium/ilxd/types"
	"time"
)

type PrivateParams struct {
	Inputs  []PrivateInput
	Outputs []PrivateOutput
}

func (priv *PrivateParams) ToExpr() (string, error) {
	inputs := ""
	for _, in := range priv.Inputs {
		input, err := in.ToExpr()
		if err != nil {
			return "", err
		}
		inputs += "(cons " + input + " "
	}
	inputs += "nil)"
	for i := 0; i < len(priv.Inputs)-1; i++ {
		inputs += ")"
	}
	if len(priv.Inputs) == 0 {
		inputs = "nil"
	}
	outputs := ""
	for _, out := range priv.Outputs {
		output, err := out.ToExpr()
		if err != nil {
			return "", err
		}
		outputs += "(cons " + output + " "
	}
	outputs += "nil)"
	for i := 0; i < len(priv.Outputs)-1; i++ {
		outputs += ")"
	}
	if len(priv.Outputs) == 0 {
		outputs = "nil"
	}
	return fmt.Sprintf("(cons %s %s)", inputs, outputs), nil
}

func (priv *PrivateParams) Clone() *PrivateParams {
	priv2 := &PrivateParams{
		Inputs:  make([]PrivateInput, len(priv.Inputs)),
		Outputs: make([]PrivateOutput, len(priv.Outputs)),
	}
	for i, in := range priv.Inputs {
		priv2.Inputs[i] = PrivateInput{
			ScriptHash:      in.ScriptHash.Clone(),
			Amount:          in.Amount,
			AssetID:         in.AssetID.Clone(),
			Salt:            in.Salt.Clone(),
			State:           make(types.State, len(in.State)),
			CommitmentIndex: in.CommitmentIndex,
			InclusionProof: InclusionProof{
				Hashes: make([][]byte, len(in.InclusionProof.Hashes)),
				Flags:  in.InclusionProof.Flags,
			},
			LockingFunction: in.LockingFunction,
			LockingParams:   make(types.LockingParams, len(in.LockingParams)),
			UnlockingParams: in.UnlockingParams,
		}
		for x, b := range in.State {
			priv2.Inputs[i].State[x] = make([]byte, len(b))
			copy(priv2.Inputs[i].State[x], b)
		}
		for x, b := range in.InclusionProof.Hashes {
			priv2.Inputs[i].InclusionProof.Hashes[x] = make([]byte, len(b))
			copy(priv2.Inputs[i].InclusionProof.Hashes[x], b)
		}
		for x, b := range in.LockingParams {
			priv2.Inputs[i].LockingParams[x] = make([]byte, len(b))
			copy(priv2.Inputs[i].LockingParams[x], b)
		}
	}
	for i, out := range priv.Outputs {
		priv2.Outputs[i] = PrivateOutput{
			ScriptHash: out.ScriptHash.Clone(),
			Amount:     out.Amount,
			AssetID:    out.AssetID.Clone(),
			Salt:       out.Salt.Clone(),
			State:      make(types.State, len(out.State)),
		}
		for x, b := range out.State {
			priv2.Outputs[i].State[x] = make([]byte, len(b))
			copy(priv2.Outputs[i].State[x], b)
		}
	}
	return priv2
}

type PublicParams struct {
	SigHash           types.ID
	Nullifiers        []types.Nullifier
	TXORoot           types.ID
	Fee               types.Amount
	Coinbase          types.Amount
	MintID            types.ID
	MintAmount        types.Amount
	Outputs           []PublicOutput
	Locktime          time.Time
	LocktimePrecision time.Duration
}

func (pub *PublicParams) ToExpr() (string, error) {
	nullifiers := ""
	for _, n := range pub.Nullifiers {
		nullifiers += fmt.Sprintf("(cons 0x%x ", n.Bytes())
	}
	nullifiers += "nil)"
	for i := 0; i < len(pub.Nullifiers)-1; i++ {
		nullifiers += ")"
	}
	if len(nullifiers) == 0 {
		nullifiers = "nil"
	}
	outputs := ""
	for _, o := range pub.Outputs {
		output, err := o.ToExpr()
		if err != nil {
			return "", err
		}
		outputs += "(cons " + output + " "
	}
	outputs += "nil)"
	for i := 0; i < len(pub.Outputs)-1; i++ {
		outputs += ")"
	}
	if len(outputs) == 0 {
		outputs = "nil"
	}

	expr := fmt.Sprintf("(cons 0x%x ", pub.SigHash.Bytes()) +
		"(cons " + nullifiers +
		fmt.Sprintf(" (cons 0x%x ", pub.TXORoot.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.Fee) +
		fmt.Sprintf("(cons %d ", pub.Coinbase) +
		fmt.Sprintf("(cons 0x%x ", pub.MintID.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.MintAmount) +
		"(cons " + outputs +
		fmt.Sprintf(" (cons %d ", pub.Locktime.Unix()) +
		fmt.Sprintf("(cons %d ", int64(pub.LocktimePrecision.Seconds())) +
		"nil))))))))))"

	return expr, nil
}

func (pub *PublicParams) Clone() *PublicParams {
	pub2 := &PublicParams{
		SigHash:           pub.SigHash.Clone(),
		Nullifiers:        make([]types.Nullifier, len(pub.Nullifiers)),
		TXORoot:           pub.TXORoot.Clone(),
		Fee:               pub.Fee,
		Coinbase:          pub.Coinbase,
		MintID:            pub.MintID.Clone(),
		MintAmount:        pub.MintAmount,
		Outputs:           make([]PublicOutput, len(pub.Outputs)),
		Locktime:          time.Unix(pub.Locktime.Unix(), 0),
		LocktimePrecision: pub.LocktimePrecision,
	}
	for i, n := range pub.Nullifiers {
		pub2.Nullifiers[i] = n.Clone()
	}
	for i, out := range pub.Outputs {
		pub2.Outputs[i] = PublicOutput{
			Commitment: out.Commitment.Clone(),
			CipherText: make([]byte, len(out.CipherText)),
		}
		copy(pub2.Outputs[i].CipherText, out.CipherText)
	}
	return pub2
}
