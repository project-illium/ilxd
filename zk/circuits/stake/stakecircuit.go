// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"encoding/binary"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/project-illium/ilxd/zk/scripts/timelockedmultisig"
	"time"
)

type PrivateParams struct {
	types.SpendNote
	CommitmentIndex  uint64
	InclusionProof   standard.InclusionProof
	ScriptCommitment []byte
	ScriptParams     [][]byte
	UnlockingParams  []byte
}

type PublicParams struct {
	TXORoot     []byte
	SigHash     []byte
	Amount      uint64
	Nullifier   []byte
	LockedUntil time.Time
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
	ul := types.LockingScript{
		ScriptCommitment: types.NewID(priv.ScriptCommitment),
		LockingParams:    priv.ScriptParams,
	}
	spendScriptHash, err := ul.Hash()
	if err != nil {
		return false
	}
	priv.ScriptHash = spendScriptHash
	priv.Amount = types.Amount(pub.Amount)

	outputCommitment, err := priv.Commitment()
	if err != nil {
		return false
	}

	// Then validate the merkle proof
	if !standard.ValidateInclusionProof(outputCommitment.Bytes(), priv.CommitmentIndex, priv.InclusionProof.Hashes, priv.InclusionProof.Flags, pub.TXORoot) {
		return false
	}

	// If the locktime is anything other than zero we need to:
	if pub.LockedUntil.After(time.Unix(0, 0)) {

		// Verify that the input script is a TimelockedMultisig script.
		if !bytes.Equal(priv.ScriptCommitment, timelockedmultisig.MockTimelockedMultisigScriptCommitment) {
			return false
		}

		if len(priv.ScriptParams) < 2 {
			return false
		}

		// Verify that the locktime used by the script is the same as the one found
		// in the body of the stake transaction.
		locktime := int64(binary.BigEndian.Uint64(priv.ScriptParams[1]))
		if !pub.LockedUntil.Equal(time.Unix(locktime, 0)) {
			return false
		}
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
	nullifier, err := types.CalculateNullifier(priv.CommitmentIndex, priv.Salt, priv.ScriptCommitment, priv.ScriptParams...)
	if err != nil {
		return false
	}
	return bytes.Equal(nullifier.Bytes(), pub.Nullifier)
}

// ValidateUnlockingScript is a placeholder. Normally this would be part of the overall circuit to validate
// the functional commitment.
func ValidateUnlockingScript(scriptCommitment []byte, scriptParams *UnlockingScriptInputs, unlockingParams []byte) (bool, error) {
	return true, nil
}
