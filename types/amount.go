// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import "encoding/binary"

const NanosPerILX = 1e9

// Amount represents the base illium monetary unit (nanoillium or nanos).
// The total number of nanoillium issued is not expected to overflow an uint64
// for 250 years from genesis.
type Amount uint64

// ToBytes returns the byte representation of the amount
func (a Amount) ToBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(a))
	return b
}

// ToILX returns the amount, formatted as ILX
func (a Amount) ToILX() float64 {
	return float64(a) / NanosPerILX
}
