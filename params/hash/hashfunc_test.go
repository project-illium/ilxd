// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package hash

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestHashFunc(t *testing.T) {
	b := make([]byte, 32)
	rand.Read(b)
	h := HashWithIndex(b, 8)
	fmt.Println(h[:])
}
