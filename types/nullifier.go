// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"fmt"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/zk"
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
	*n = i
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
	ul := LockingScript{
		ScriptCommitment: NewID(scriptCommitment),
		LockingParams:    scriptParams,
	}
	expr, err := ul.lurkExpression()
	if err != nil {
		return Nullifier{}, err
	}

	expr = fmt.Sprintf("(cons %d (cons 0x%x ", commitmentIndex, salt) + expr + "))"
	h, err := zk.LurkCommit(expr)
	if err != nil {
		return Nullifier{}, err
	}
	return NewNullifier(h), nil
}
