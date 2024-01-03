// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"encoding/hex"
	"fmt"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProve(t *testing.T) {
	LoadZKPublicParameters()

	program := `
(lambda (priv pub) 
	(letrec ((cat-and-hash (lambda (a b)
		    (eval (cons '.lurk.blake2s (cons a (cons b nil)))))))
	  (= (num (cat-and-hash 0x1ab158775ec343edacd4aa9f395a8ab61cfd544b6dc260a44374efac8457a874 0x18b1a7da2e8bc9c7633224d4df95f5730e521bbc6d57c8db9ab51a2f963d703b)) 0x07998a4da2077a58a4d9b0c01cfaec351dfd19c9e064ab667ce6b4b7fe1e7bfa)))
`

	start := time.Now()
	proof, err := Prove(program, Expr("nil"), Expr("nil"))
	assert.NoError(t, err)
	fmt.Println(len(proof))
	fmt.Println(time.Since(start))
}

func TestBasicTransferScript(t *testing.T) {
	l, err := hex.DecodeString("1ab158775ec343edacd4aa9f395a8ab61cfd544b6dc260a44374efac8457a874")
	assert.NoError(t, err)
	r, err := hex.DecodeString("18b1a7da2e8bc9c7633224d4df95f5730e521bbc6d57c8db9ab51a2f963d703b")
	assert.NoError(t, err)
	o := hash.HashMerkleBranches(r, l)
	fmt.Println(hex.EncodeToString(o))
}
