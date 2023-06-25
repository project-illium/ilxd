// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/project-illium/ilxd/params/hash"
)

const NullifierSize = hash.HashSize

type Nullifier [hash.HashSize]byte

func (n Nullifier) String() string {
	return hex.EncodeToString(n[:])
}

func (n Nullifier) Bytes() []byte {
	return n[:]
}

func (n Nullifier) Clone() Nullifier {
	var b [len(n)]byte
	copy(b[:], n[:])
	return b
}

func (n *Nullifier) SetBytes(data []byte) {
	copy(n[:], data)
}

func (n *Nullifier) MarshalJSON() ([]byte, error) {
	return []byte(hex.EncodeToString(n[:])), nil
}

func (n *Nullifier) UnmarshalJSON(data []byte) error {
	i, err := NewNullifierFromString(string(data)) //nolint:staticcheck
	if err != nil {
		return err
	}
	n = &i //nolint:ineffassign
	return nil
}

func NewNullifier(b []byte) Nullifier {
	var sh Nullifier
	sh.SetBytes(b)
	return sh
}

func NewNullifierFromString(n string) (Nullifier, error) {
	// Return error if hash string is too long.
	if len(n) > hash.HashSize*2 {
		return Nullifier{}, ErrIDStrSize
	}
	ret, err := hex.DecodeString(n)
	if err != nil {
		return Nullifier{}, err
	}
	var newN Nullifier
	newN.SetBytes(ret)
	return newN, nil
}

// CalculateNullifier calculates and returns the nullifier for the given inputs.
func CalculateNullifier(commitmentIndex uint64, salt [32]byte, scriptCommitment []byte, scriptParams ...[]byte) (Nullifier, error) {
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, commitmentIndex)

	ser := make([]byte, 0, 8+32+32+(len(scriptParams)*32))

	ser = append(ser, indexBytes...)
	ser = append(ser, salt[:]...)
	ser = append(ser, scriptCommitment...)
	for _, params := range scriptParams {
		ser = append(ser, params...)
	}
	h := hash.HashFunc(ser)
	var out [32]byte
	copy(out[:], h)
	return out, nil
}
