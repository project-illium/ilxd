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
	AssetID              [32]byte
	Salt                 [32]byte
	State                [32]byte
	CommitmentIndex      uint64
	InclusionProof       standard.InclusionProof
	SnarkVerificationKey []byte
	UserParams           [][]byte
	SnarkProof           []byte
}

type PublicParams struct {
	TXORoot   []byte
	SigHash   []byte
	Amount    uint64
	Nullifier []byte
	Locktime  time.Time
}

type UnlockingSnarkParams struct {
	InputIndex    int
	PrivateParams PrivateParams
	PublicParams  PublicParams
	UserParams    [][]byte
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
	spendScriptPreimage := priv.SnarkVerificationKey
	for _, param := range priv.UserParams {
		spendScriptPreimage = append(spendScriptPreimage, param...)
	}
	spendScriptHash := hash.HashFunc(spendScriptPreimage)

	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, pub.Amount)
	commitmentPreimage := make([]byte, 0, 32+8+32+32+32)
	commitmentPreimage = append(commitmentPreimage, spendScriptHash...)
	commitmentPreimage = append(commitmentPreimage, amountBytes...)
	commitmentPreimage = append(commitmentPreimage, priv.AssetID[:]...)
	commitmentPreimage = append(commitmentPreimage, priv.State[:]...)
	commitmentPreimage = append(commitmentPreimage, priv.Salt[:]...)
	outputCommitment := hash.HashFunc(commitmentPreimage)

	// Then validate the merkle proof
	if !standard.ValidateInclusionProof(outputCommitment, priv.CommitmentIndex, priv.InclusionProof.Hashes, priv.InclusionProof.Flags, priv.InclusionProof.Accumulator, pub.TXORoot) {
		return false
	}

	// Validate the unlocking snark.
	unlockingParams := &UnlockingSnarkParams{
		InputIndex:    0,
		PrivateParams: *priv,
		PublicParams:  *pub,
		UserParams:    priv.UserParams,
	}

	valid, err := ValidateUnlockingSnark(priv.SnarkVerificationKey, unlockingParams, priv.SnarkProof)
	if !valid || err != nil {
		return false
	}

	// Validate that the nullifier is calculated correctly.
	commitmentIndexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(commitmentIndexBytes, priv.CommitmentIndex)
	nullifierPreimage := make([]byte, 0, 8+32+100)
	nullifierPreimage = append(nullifierPreimage, commitmentIndexBytes...)
	nullifierPreimage = append(nullifierPreimage, priv.Salt[:]...)
	nullifierPreimage = append(nullifierPreimage, priv.SnarkVerificationKey...)
	for _, param := range priv.UserParams {
		nullifierPreimage = append(nullifierPreimage, param...)
	}
	calculatedNullifier := hash.HashFunc(nullifierPreimage)
	if !bytes.Equal(calculatedNullifier, pub.Nullifier) {
		return false
	}

	return true
}

// ValidateUnlockingSnark is a placeholder. Normally this would be part of the overall circuit to validate
// the inner snark recursively.
func ValidateUnlockingSnark(snarkVerificationKey []byte, publicParams *UnlockingSnarkParams, proof []byte) (bool, error) {
	return true, nil
}
