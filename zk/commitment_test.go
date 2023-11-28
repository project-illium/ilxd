// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLurkCommit(t *testing.T) {
	h, err := LurkCommit("(cons 1 (cons 2 nil))")
	assert.NoError(t, err)
	fmt.Println(h)
}
