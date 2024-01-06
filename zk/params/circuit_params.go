// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

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
		output += "(cons " + output + " "
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

type PublicParams struct {
	Nullifiers        []types.Nullifier
	TXORoot           types.ID
	Fee               types.Amount
	Coinbase          types.Amount
	MintID            types.ID
	MintAmount        types.Amount
	Outputs           []PublicOutput
	SigHash           types.ID
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

	expr := "(cons " + nullifiers +
		fmt.Sprintf(" (cons 0x%x ", pub.TXORoot.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.Fee) +
		fmt.Sprintf("(cons %d ", pub.Coinbase) +
		fmt.Sprintf("(cons 0x%x ", pub.MintID.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.MintAmount) +
		"(cons " + outputs +
		fmt.Sprintf(" (cons 0x%x ", pub.SigHash.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.Locktime.Unix()) +
		fmt.Sprintf("(cons %d ", int64(pub.LocktimePrecision.Seconds())) +
		"nil))))))))))"

	return expr, nil
}
