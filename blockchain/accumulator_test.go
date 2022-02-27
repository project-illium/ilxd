// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"
)

// FIXME
func TestAccumulator(t *testing.T) {
	data := make([][]byte, 100000)
	for i := range data {
		data[i] = make([]byte, 5000)
		rand.Read(data[i])
	}
	start := time.Now()
	acc := NewAccumulator()
	for _, d := range data {
		acc.Insert(d, false)
	}
	root := acc.Root()
	fmt.Println(root.String())
	fmt.Println(time.Since(start))
}
