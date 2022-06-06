// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"encoding/binary"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
)

var IlliumCoinID = types.NewID(bytes.Repeat([]byte{0x00}, 32))

const CommitmentLen = 32

// SpendNote holds all the data that makes up an output commitment.
type SpendNote struct {
	SpendScript SpendScript
	Amount      uint64
	AssetID     types.ID
	Salt        [32]byte
}

// Commitment serializes and hashes the data in the note and
// returns the hash.
func (s *SpendNote) Commitment() ([]byte, error) {
	spendScript, err := s.SpendScript.Hash()
	if err != nil {
		return nil, err
	}

	ser := make([]byte, 0, 32+8+32+32)

	idBytes := s.AssetID.Bytes()
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, s.Amount)

	ser = append(ser, spendScript...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, s.Salt[:]...)

	return hash.HashFunc(ser), nil
}
