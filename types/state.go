// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
)

// State represents the state field out an output note.
//
// There are two different representations of the state:
//   - Inside a lurk program it is a list expression of field elements
//   - Outside a lurk program we serialize it as a byte slice when encrypting
//     the output note.
type State [][]byte

// Serialize serializes the state into a format that can be encrypted
// and used as the ciphertext in a Transaction.
//
// Each element is prepended with the element's length. If pad is true
// and the serialized length is less than 128, zeros will be appended.
//
// A length byte of zero is treated as a delimiter.
func (s *State) Serialize(pad bool) ([]byte, error) {
	ser := serializeData(*s)
	if pad {
		ser = append(ser, bytes.Repeat([]byte{0x00}, 128-len(ser))...)
	}
	return ser, nil
}

// Deserialize will deserialize the slice into the State.
func (s *State) Deserialize(ser []byte) error {
	state, err := deserializeData(ser)
	if err != nil {
		return err
	}
	*s = state
	return nil
}

// ToLurkExpression converts the state to a lurk expression for use
// inside the lurk program.
//
// Boolean:
// Elements with a len of 1 will be treated as a bool.
//
// Uint64:
// Elements with len of 2-8 will be treated as a uint64.
//
// Num:
// Elements with a len of 9-31 will be treated as a big endian lurk num.
//
// Hash:
// If the element is len 32 it will be treated as a hex string prefixed
// with 0x. Lurk interprets this the same as Num.
func (s *State) ToLurkExpression() (string, error) {
	if s == nil || len(*s) == 0 {
		return "", nil
	}
	return buildLurkExpression(*s)
}
