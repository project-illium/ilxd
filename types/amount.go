// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import "encoding/binary"

// Amount represents the base illium monetary unit (to be named later).
// The total number of coins issued is not expected to overflow an uint64
// for 250 years from genesis.
type Amount uint64

// ToBytes returns the byte repesentation of the amount
func (a Amount) ToBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(a))
	return b
}
