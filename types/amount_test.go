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

	assert.Equal(t, a, AmountFromILX("230584300.921369395"))

	j, err := a.MarshalJSON()
	assert.NoError(t, err)

	assert.Equal(t, "230584300.921369395", string(j))

	var a2 Amount
	err = a2.UnmarshalJSON(j)
	assert.NoError(t, err)

	assert.Equal(t, a, a2)
}
