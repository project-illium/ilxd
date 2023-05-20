// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGo(t *testing.T) {
	a, err := hex.DecodeString("b178667bdbe651d9540bcc21e90a71360a474a5883b9eecc552004b75a681ec8")
	assert.NoError(t, err)
	h := types.HexEncodable(a)

	m, err := json.Marshal(h)
	assert.NoError(t, err)
	fmt.Println("***", string(m))

	var h2 types.HexEncodable
	err = json.Unmarshal(m, &h2)
	assert.NoError(t, err)

	fmt.Println("$$$", hex.EncodeToString(h2))
}
