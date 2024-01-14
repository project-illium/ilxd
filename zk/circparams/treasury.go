// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package circparams

import (
	"fmt"
	"github.com/project-illium/ilxd/types"
)

type TreasuryPrivateParams []PrivateOutput

func (priv *TreasuryPrivateParams) ToExpr() (string, error) {
	outputs := ""
	for _, o := range *priv {
		output, err := o.ToExpr()
		if err != nil {
			return "", err
		}
		outputs += "(cons " + output + " "
	}
	outputs += "nil)"
	for i := 0; i < len(*priv)-1; i++ {
		outputs += ")"
	}
	if len(*priv) == 0 {
		outputs = "nil"
	}
	return outputs, nil
}

type TreasuryPublicParams struct {
	TreasuryWithdrawAmount types.Amount
	Outputs                []PublicOutput
}

func (pub *TreasuryPublicParams) ToExpr() (string, error) {
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
	if len(pub.Outputs) == 0 {
		outputs = "nil"
	}

	expr := fmt.Sprintf("(cons %d ", pub.TreasuryWithdrawAmount) +
		"(cons " + outputs +
		" nil))"

	return expr, nil
}
