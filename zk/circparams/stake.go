// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package circparams

import (
	"fmt"
	"github.com/project-illium/ilxd/types"
	"time"
)

type StakePrivateParams PrivateParams

type StakePublicParams1 struct {
	StakeAmount types.Amount
	SigHash     types.ID
	Nullifier   types.Nullifier
	TXORoot     types.ID
	LockedUntil time.Time
}

func (pub *StakePublicParams1) ToExpr() (string, error) {
	expr := fmt.Sprintf("(cons %d ", pub.StakeAmount) +
		fmt.Sprintf("(cons 0x%x ", pub.SigHash.Bytes()) +
		fmt.Sprintf("(cons 0x%x ", pub.Nullifier.Bytes()) +
		fmt.Sprintf(" (cons 0x%x ", pub.TXORoot.Bytes()) +
		fmt.Sprintf(" (cons %d ", pub.LockedUntil.Unix()) +
		"nil)))))"

	return expr, nil
}
