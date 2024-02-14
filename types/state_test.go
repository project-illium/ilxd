// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState(t *testing.T) {
	param2, err := hex.DecodeString("0cef7dd85c04c505d55c063824a5bad62170db0d37e2068fc6c749ada2cb8293")
	assert.NoError(t, err)
	param1 := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	state := State{param1, param2}

	expr, err := state.ToExpr()
	assert.NoError(t, err)

	assert.Equal(t, "(cons 1 (cons 0x0cef7dd85c04c505d55c063824a5bad62170db0d37e2068fc6c749ada2cb8293 nil))", expr)

	ser, err := state.Serialize(false)
	assert.NoError(t, err)

	assert.Len(t, ser, 9+33)

	ser, err = state.Serialize(true)
	assert.NoError(t, err)

	assert.Len(t, ser, 128)

	state2 := new(State)
	err = state2.Deserialize(ser)
	assert.NoError(t, err)

	assert.Equal(t, state, *state2)
}
