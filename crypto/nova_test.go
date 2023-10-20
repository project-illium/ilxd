// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestNewSecretKey(t *testing.T) {
	key := NewSecretKey()
	fmt.Println(hex.EncodeToString(key))
}
