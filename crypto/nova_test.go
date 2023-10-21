// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestNewSecretKey(t *testing.T) {
	sk := NewSecretKey()
	fmt.Println(hex.EncodeToString(sk))

	fmt.Println("***", len(sk))

	pk := PrivToPub(sk)
	fmt.Println(hex.EncodeToString(pk))

	fmt.Println("***", len(pk))

	b := make([]byte, 32)
	rand.Read(b)

	sig := Sign(sk, b)
	fmt.Println(hex.EncodeToString(sig))

	fmt.Println("***", len(sig))

	valid := Verify(pk, b, sig)
	fmt.Println(valid)
}
