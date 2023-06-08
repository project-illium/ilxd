// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/project-illium/ilxd/params/hash"
)

var IlliumCoinID = NewID(bytes.Repeat([]byte{0x00}, 32))

const (
	CommitmentLen = 32
	ScriptHashLen = 32
	AmountLen     = 8
	AssetIDLen    = 32
	StateLen      = 128
	SaltLen       = 32
)

type UnlockingScript struct {
	ScriptCommitment []byte
	ScriptParams     [][]byte
}

func (u *UnlockingScript) Serialize() []byte {
	ser := make([]byte, len(u.ScriptCommitment))
	copy(ser, u.ScriptCommitment)
	for _, param := range u.ScriptParams {
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
	ScriptHash []byte
	Amount     Amount
	AssetID    ID
	State      [StateLen]byte
	Salt       [SaltLen]byte
}

// Commitment serializes and hashes the data in the note and
// returns the hash.
func (s *SpendNote) Commitment() ([]byte, error) {
	ser := s.Serialize()
	return hash.HashFunc(ser), nil
}

func (s *SpendNote) Serialize() []byte {
	ser := make([]byte, 0, ScriptHashLen+AmountLen+AssetIDLen+StateLen+SaltLen)

	idBytes := s.AssetID.Bytes()
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, uint64(s.Amount))

	ser = append(ser, s.ScriptHash...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, s.State[:]...)
	ser = append(ser, s.Salt[:]...)
	return ser
}

func (s *SpendNote) Deserialize(ser []byte) error {
	if len(ser) != ScriptHashLen+AmountLen+AssetIDLen+StateLen+SaltLen {
		return errors.New("invalid serialization length")
	}

	copy(s.ScriptHash, ser[:ScriptHashLen])
	s.Amount = Amount(binary.BigEndian.Uint64(ser[ScriptHashLen : ScriptHashLen+AmountLen]))
	copy(s.AssetID[:], ser[ScriptHashLen+AmountLen:ScriptHashLen+AmountLen+AssetIDLen])
	copy(s.State[:], ser[ScriptHashLen+AmountLen+AssetIDLen:ScriptHashLen+AmountLen+AssetIDLen+StateLen])
	copy(s.Salt[:], ser[ScriptHashLen+AmountLen+AssetIDLen+StateLen:])
	return nil
}
