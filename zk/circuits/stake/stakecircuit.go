// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"encoding/binary"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"time"
)

type PrivateParams struct {
	Amount          uint64
	AssetID         []byte
	Salt            []byte
	CommitmentIndex uint64
	InclusionProof  standard.InclusionProof
	Threshold       uint8
	Pubkeys         [][]byte
	Signatures      [][]byte
	SigBitfield     uint8
}

type PublicParams struct {
	TXORoot   []byte
	SigHash   []byte
	Amount    uint64
	Nullifier []byte
	Blocktime time.Time
}

func StakeCircuit(privateParams, publicParams interface{}) bool {
	priv, ok := privateParams.(*PrivateParams)
	if !ok {
		return false
	}
	pub, ok := publicParams.(*PublicParams)
	if !ok {
		return false
	}

	// First obtain the hash of the spendScript.
	spendScriptPreimage := []byte{priv.Threshold}
	for _, key := range priv.Pubkeys {
		spendScriptPreimage = append(spendScriptPreimage, key...)
	}
	spendScriptHash := hash.HashFunc(spendScriptPreimage)

	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, priv.Amount)
	commitmentPreimage := make([]byte, 0, 32+8+32+32)
	commitmentPreimage = append(commitmentPreimage, spendScriptHash...)
	commitmentPreimage = append(commitmentPreimage, amountBytes...)
	commitmentPreimage = append(commitmentPreimage, priv.AssetID[:]...)
	commitmentPreimage = append(commitmentPreimage, priv.Salt...)
	outputCommitment := hash.HashFunc(commitmentPreimage)

	// Then validate the merkle proof
	if !standard.ValidateInclusionProof(outputCommitment, priv.CommitmentIndex, priv.InclusionProof.Hashes, priv.InclusionProof.Flags, priv.InclusionProof.Accumulator, pub.TXORoot) {
		return false
	}

	// Validate the signature(s)
	valid, err := standard.ValidateMultiSignature(priv.Threshold, priv.Pubkeys, priv.Signatures, priv.SigBitfield, pub.SigHash, pub.Blocktime)
	if !valid || err != nil {
		return false
	}

	// Validate that the nullifier is calculated correctly.
	commitmentIndexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(commitmentIndexBytes, priv.CommitmentIndex)
	nullifierPreimage := make([]byte, 0, 8+32+1+(len(priv.Pubkeys)*32))
	nullifierPreimage = append(nullifierPreimage, commitmentIndexBytes...)
	nullifierPreimage = append(nullifierPreimage, priv.Salt...)
	nullifierPreimage = append(nullifierPreimage, priv.Threshold)
	for _, key := range priv.Pubkeys {
		nullifierPreimage = append(nullifierPreimage, key...)
	}
	calculatedNullifier := hash.HashFunc(nullifierPreimage)
	if !bytes.Equal(calculatedNullifier, pub.Nullifier) {
		return false
	}

	if priv.Amount != pub.Amount {
		return false
	}

	return true
}
