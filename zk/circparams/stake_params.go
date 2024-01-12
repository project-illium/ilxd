// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package circparams

import (
	"fmt"
	"github.com/project-illium/ilxd/types"
)

type StakePublicParams struct {
	StakeAmount types.Amount
	PublicParams
}

func (s *StakePublicParams) ToExpr() (string, error) {
	pubExpr, err := s.PublicParams.ToExpr()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(cons %d (cons %s nil))", s.StakeAmount, pubExpr), nil
}
