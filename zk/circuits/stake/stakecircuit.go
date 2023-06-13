// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"encoding/binary"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"time"
)

type PrivateParams struct {
	AssetID          [types.AssetIDLen]byte
	Salt             [types.SaltLen]byte
	State            [types.StateLen]byte
	CommitmentIndex  uint64
	InclusionProof   standard.InclusionProof
	ScriptCommitment []byte
	ScriptParams     [][]byte
	UnlockingParams  [][]byte
}

type PublicParams struct {
	TXORoot   []byte
	SigHash   []byte
	Amount    uint64
	Nullifier []byte
	Locktime  time.Time
}

type UnlockingScriptInputs struct {
	InputIndex    int
	PrivateParams PrivateParams
	PublicParams  PublicParams
	ScriptParams  [][]byte
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
	spendScriptPreimage := priv.ScriptCommitment
	for _, param := range priv.ScriptParams {
		spendScriptPreimage = append(spendScriptPreimage, param...)
	}
	spendScriptHash := hash.HashFunc(spendScriptPreimage)

	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, pub.Amount)
	commitmentPreimage := make([]byte, 0, hash.HashSize+8+types.AssetIDLen+types.StateLen+types.SaltLen)
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
	unlockingParams := &UnlockingScriptInputs{
		InputIndex:    0,
		PrivateParams: *priv,
		PublicParams:  *pub,
		ScriptParams:  priv.ScriptParams,
	}

	valid, err := ValidateUnlockingScript(priv.ScriptCommitment, unlockingParams, priv.UnlockingParams)
	if !valid || err != nil {
		return false
	}

	// Validate that the nullifier is calculated correctly.
	commitmentIndexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(commitmentIndexBytes, priv.CommitmentIndex)
	nullifierPreimage := make([]byte, 0, 8+32+100)
	nullifierPreimage = append(nullifierPreimage, commitmentIndexBytes...)
	nullifierPreimage = append(nullifierPreimage, priv.Salt[:]...)
	nullifierPreimage = append(nullifierPreimage, priv.ScriptCommitment...)
	for _, param := range priv.ScriptParams {
		nullifierPreimage = append(nullifierPreimage, param...)
	}
	calculatedNullifier := hash.HashFunc(nullifierPreimage)
	return bytes.Equal(calculatedNullifier, pub.Nullifier)
}

// ValidateUnlockingScript is a placeholder. Normally this would be part of the overall circuit to validate
// the functional commitment.
func ValidateUnlockingScript(scriptCommitment []byte, scriptParams *UnlockingScriptInputs, unlockingParams [][]byte) (bool, error) {
	return true, nil
}
