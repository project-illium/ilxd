// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/crypto"
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
	serializedSpendScript, err := o.SpendScript.Serialize()
	if err != nil {
		return nil, err
	}

	ser := make([]byte, 0, 32+8+32+32)

	idBytes := o.AssetID.Bytes()
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, o.Amount)

	ser = append(ser, hash.HashFunc(serializedSpendScript)...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, o.Salt[:]...)

	return hash.HashFunc(ser), nil
}

// CalculateNullifier calculates and returns the nullifier for the given inputs.
func CalculateNullifier(commitmentIndex uint64, salt [32]byte, threshold uint8, pubkeys ...crypto.PubKey) ([]byte, error) {
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, commitmentIndex)

	ser := make([]byte, 0, 8+32+1+(len(pubkeys)*32))

	ser = append(ser, indexBytes...)
	ser = append(ser, salt[:]...)
	ser = append(ser, threshold)
	for _, pubkey := range pubkeys {
		raw, err := pubkey.Raw()
		if err != nil {
			return nil, err
		}
		ser = append(ser, raw...)
	}
	return hash.HashFunc(ser), nil
}
