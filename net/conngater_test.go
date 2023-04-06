// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"fmt"
	"testing"
)

func TestNewBasicConnectionGater(t *testing.T) {
	s := make(map[string]*DynamicBanScore)

	s["a"] = &DynamicBanScore{}

	score := s["a"]

	score.Increase(10, 10)

	fmt.Println(s["a"])
}
