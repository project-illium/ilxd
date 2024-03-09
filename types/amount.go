// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

const NanosPerILX = 1e9

// Amount represents the base illium monetary unit (nanoillium or nanos).
// The total number of nanoillium issued is not expected to overflow an uint64
// for 250 years from genesis.
type Amount uint64

// AmountFromILX returns an amount (in nanoillium) from an ILX float
func AmountFromILX(ilx float64) Amount {
	return Amount(ilx * NanosPerILX)
}

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

// MarshalJSON returns the amount in ILX as a float string
func (a *Amount) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.9f", a.ToILX())), nil
}

// UnmarshalJSON unmarshals an ILX float string to an Amount.
func (a *Amount) UnmarshalJSON(data []byte) error {
	str := string(data)
	str = str[1 : len(str)-1]

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	*a = Amount(val * NanosPerILX)
	return nil
}
