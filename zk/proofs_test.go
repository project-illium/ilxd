// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProve(t *testing.T) {
	LoadZKPublicParameters()

	r, err := randomFieldElement()
	assert.NoError(t, err)

	h, err := LurkCommit(fmt.Sprintf("0x%x", r[:]))
	assert.NoError(t, err)

	program := "(lambda (priv pub) (= (num (commit priv)) pub))"
	start := time.Now()
	proof, err := Prove(program, Expr(fmt.Sprintf("0x%x", r[:])), Expr(fmt.Sprintf("0x%x", h[:])))
	assert.NoError(t, err)
	fmt.Println(len(proof))
	fmt.Println(time.Since(start))
}
