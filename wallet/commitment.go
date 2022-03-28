// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/binary"
	"github.com/project-illium/ilxd/models"
	"github.com/project-illium/ilxd/params/hash"
)

// OutputCommitmentPreimage holds all the data that makes up an output
// commitment.
type OutputCommitmentPreimage struct {
	SpendScript SpendScript
	Amount      uint64
	AssetID     models.ID
	Salt        [32]byte
}

// Commitment serializes and hashes the data in the preimage and
// returns the hash.
func (o *OutputCommitmentPreimage) Commitment() ([]byte, error) {
	spendScript, err := o.SpendScript.Hash()
	if err != nil {
		return nil, err
	}

	ser := make([]byte, 0, 32+8+32+32)

	idBytes := o.AssetID.Bytes()
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, o.Amount)

	ser = append(ser, spendScript...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, o.Salt[:]...)

	return hash.HashFunc(ser), nil
}
