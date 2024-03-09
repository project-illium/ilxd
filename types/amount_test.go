// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAmountJsonMarshaling(t *testing.T) {
	a := Amount(230584300921369395)

	assert.Equal(t, 230584300.921369395, a.ToILX())

	a2, err := AmountFromILX("230584300.921369395")
	assert.NoError(t, err)
	assert.Equal(t, a, a2)

	j, err := a.MarshalJSON()
	assert.NoError(t, err)

	assert.Equal(t, "230584300.921369395", string(j))

	var a3 Amount
	err = a3.UnmarshalJSON(j)
	assert.NoError(t, err)

	assert.Equal(t, a, a3)
}
