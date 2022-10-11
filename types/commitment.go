// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/binary"
	"github.com/project-illium/ilxd/params/hash"
)

var IlliumCoinID = NewID(bytes.Repeat([]byte{0x00}, 32))

const CommitmentLen = 32

type UnlockingScript struct {
	SnarkVerificationKey []byte
	PublicParams         [][]byte
}

func (u *UnlockingScript) Serialize() []byte {
	ser := make([]byte, len(u.SnarkVerificationKey))
	copy(ser, u.SnarkVerificationKey)
	for _, param := range u.PublicParams {
		p := make([]byte, len(param))
		copy(p, param)
		ser = append(ser, p...)
	}
	return ser
}

func (u *UnlockingScript) Hash() ID {
	ser := u.Serialize()
	return NewIDFromData(ser)
}

// SpendNote holds all the data that makes up an output commitment.
type SpendNote struct {
	UnlockingScript UnlockingScript
	Amount          uint64
	AssetID         ID
	State           [32]byte
	Salt            [32]byte
}

// Commitment serializes and hashes the data in the note and
// returns the hash.
func (s *SpendNote) Commitment() ([]byte, error) {
	scriptHash := s.UnlockingScript.Hash()

	ser := make([]byte, 0, 32+8+32+32+32)

	idBytes := s.AssetID.Bytes()
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, s.Amount)

	ser = append(ser, scriptHash[:]...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, s.State[:]...)
	ser = append(ser, s.Salt[:]...)

	return hash.HashFunc(ser), nil
}
