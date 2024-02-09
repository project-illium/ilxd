// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"bytes"
	"fmt"
)

// VerifyInputScript is a function that can be used to test locking scripts.
// You provide the script and the set of parameters as they would look in
// a transaction and the script will be executed.
//
// Example:
// script := "(lambda (a b c d e) (= c 10))"
// valid, err := VerifyInputScript(script, Expr("nil"), Expr("nil"), 10, Expr("nil"), Expr("nil"))
func VerifyInputScript(script string, lockingParams Parameters, unlockingParams Parameters, inputIndex uint32, privateParams Parameters, publicParams Parameters) (bool, error) {
	lp, err := lockingParams.ToExpr()
	if err != nil {
		return false, err
	}
	ulp, err := unlockingParams.ToExpr()
	if err != nil {
		return false, err
	}
	priv, err := privateParams.ToExpr()
	if err != nil {
		return false, err
	}
	pub, err := publicParams.ToExpr()
	if err != nil {
		return false, err
	}
	program := fmt.Sprintf(`(lambda (priv pub) 
					(letrec ((script %s))
						(script %s %s %d %s %s)))`, script, lp, ulp, inputIndex, priv, pub)

	tag, val, _, err := Eval(program, Expr("nil"), Expr("nil"))
	if err != nil {
		return false, err
	}

	return tag == TagSym && bytes.Equal(val, OutputTrue), nil
}
