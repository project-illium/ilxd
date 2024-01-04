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

	program := "(lambda (priv pub) (= priv pub))"

	start := time.Now()
	proof, err := Prove(program, Expr("1"), Expr("1"))
	assert.NoError(t, err)
	fmt.Println(len(proof))
	fmt.Println(time.Since(start))

	start = time.Now()
	valid, err := Verify(program, Expr("1"), proof)
	assert.NoError(t, err)
	assert.True(t, valid)
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
