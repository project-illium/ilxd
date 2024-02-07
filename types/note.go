// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/binary"
	"errors"
	"github.com/project-illium/ilxd/zk"
)

// SpendNote holds all the data that makes up an output commitment.
type SpendNote struct {
	ScriptHash ID
	Amount     Amount
	AssetID    ID
	Salt       [SaltLen]byte
	State      State
}

// Commitment builds a Lurk list expression out of the note
// data and returns the Lurk Commitment hash.
func (s *SpendNote) Commitment() (ID, error) {
	elems := []any{
		s.ScriptHash.Bytes(),
		s.Amount.ToBytes(),
		s.AssetID.Bytes(),
		s.Salt[:],
	}
	stateExpr, err := s.State.ToExpr()
	if err != nil {
		return ID{}, err
	}
	elems = append(elems, stateExpr)
	expr, err := buildLurkExpression(elems)
	if err != nil {
		return ID{}, err
	}
	h, err := zk.LurkCommit(expr)
	if err != nil {
		return ID{}, err
	}
	return NewID(h), nil
}

// Serialize returns the note serialized as a byte array. This format is suitable
// for encrypting and including in a transaction output.
func (s *SpendNote) Serialize() ([]byte, error) {
	ser := make([]byte, 0, ScriptHashLen+AmountLen+AssetIDLen+StateLen+SaltLen)

	idBytes := s.AssetID.Bytes()
	amountBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(amountBytes[0:8], uint64(s.Amount))

	ser = append(ser, s.ScriptHash.Bytes()...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, s.Salt[:]...)

	stateSer, err := s.State.Serialize(true)
	if err != nil {
		return nil, err
	}
	ser = append(ser, stateSer...)
	return ser, nil
}

// Deserialize turns a serialized byte slice back into a SpendNote
func (s *SpendNote) Deserialize(ser []byte) error {
	if len(ser) < ScriptHashLen+AmountLen+AmountPad+AssetIDLen+SaltLen {
		return errors.New("invalid serialization length")
	}
	copy(s.ScriptHash[:], ser[:ScriptHashLen])

	s.Amount = Amount(binary.BigEndian.Uint64(ser[ScriptHashLen : ScriptHashLen+AmountLen]))
	copy(s.AssetID[:], ser[ScriptHashLen+AmountLen+AmountPad:ScriptHashLen+AmountLen+AmountPad+AssetIDLen])
	copy(s.Salt[:], ser[ScriptHashLen+AmountLen+AmountPad+AssetIDLen:ScriptHashLen+AmountLen+AmountPad+AssetIDLen+SaltLen])

	state := &State{}
	if err := state.Deserialize(ser[ScriptHashLen+AmountLen+AmountPad+AssetIDLen+SaltLen:]); err != nil {
		return err
	}
	s.State = *state
	return nil
}
