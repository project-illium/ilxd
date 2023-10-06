// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package standard

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"math"
	"time"
)

var ErrIntegerOverflow = errors.New("integer overflow")

var (
	defaultAssetID [types.AssetIDLen]byte
)

type InclusionProof struct {
	Hashes      [][]byte
	Flags       uint64
	Accumulator [][]byte
}

type PrivateInput struct {
	Amount           uint64
	Salt             [types.SaltLen]byte
	AssetID          [types.AssetIDLen]byte
	State            [types.StateLen]byte
	CommitmentIndex  uint64
	InclusionProof   InclusionProof
	ScriptCommitment []byte
	ScriptParams     [][]byte
	UnlockingParams  [][]byte
}

type PrivateOutput struct {
	ScriptHash []byte
	Amount     uint64
	Salt       [types.SaltLen]byte
	AssetID    [types.AssetIDLen]byte
	State      [types.StateLen]byte
}

type PublicOutput struct {
	Commitment []byte
	CipherText []byte
}

type PrivateParams struct {
	Inputs  []PrivateInput
	Outputs []PrivateOutput
}

type PublicParams struct {
	TXORoot             []byte
	SigHash             []byte
	Outputs             []PublicOutput
	Nullifiers          [][]byte
	Fee                 uint64
	Coinbase            uint64
	MintID              []byte
	MintAmount          uint64
	Locktime            time.Time
	LocktimeGranularity time.Duration
}

type UnlockingScriptInputs struct {
	InputIndex    int
	PrivateParams PrivateParams
	PublicParams  PublicParams
	ScriptParams  [][]byte
}

// This whole function is a placeholder for the actual zk-snark circuit. We enumerate it
// here to give an approximate idea of what the circuit will do.
func StandardCircuit(privateParams, publicParams interface{}) bool {
	priv, ok := privateParams.(*PrivateParams)
	if !ok {
		return false
	}
	pub, ok := publicParams.(*PublicParams)
	if !ok {
		return false
	}
	var (
		inVal = uint64(0)
		err   error
	)
	assetIns := make(map[[types.AssetIDLen]byte]uint64)

	for i, in := range priv.Inputs {
		// First obtain the hash of the spendScript.
		spendScriptPreimage := in.ScriptCommitment
		for _, param := range in.ScriptParams {
			spendScriptPreimage = append(spendScriptPreimage, param...)
		}
		spendScriptHash := hash.HashFunc(spendScriptPreimage)

		// Now calculate the commitmentPreimage and commitment hash.
		amountBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(amountBytes, in.Amount)
		commitmentPreimage := make([]byte, 0, hash.HashSize+8+types.AssetIDLen+types.StateLen+types.SaltLen)
		commitmentPreimage = append(commitmentPreimage, spendScriptHash...)
		commitmentPreimage = append(commitmentPreimage, amountBytes...)
		commitmentPreimage = append(commitmentPreimage, in.AssetID[:]...)
		commitmentPreimage = append(commitmentPreimage, in.State[:]...)
		commitmentPreimage = append(commitmentPreimage, in.Salt[:]...)

		outputCommitment := hash.HashFunc(commitmentPreimage)

		// Then validate the merkle proof using the calculated commitment hash
		// and provided inclusion proof.
		if !ValidateInclusionProof(outputCommitment, in.CommitmentIndex, in.InclusionProof.Hashes, in.InclusionProof.Flags, in.InclusionProof.Accumulator, pub.TXORoot) {
			return false
		}

		// Validate the unlocking snark.
		unlockingParams := &UnlockingScriptInputs{
			InputIndex:    i,
			PrivateParams: *priv,
			PublicParams:  *pub,
			ScriptParams:  in.ScriptParams,
		}

		valid, err := ValidateUnlockingScript(in.ScriptCommitment, unlockingParams, in.UnlockingParams)
		if !valid || err != nil {
			return false
		}

		// Validate that the nullifier is calculated correctly.
		commitmentIndexBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(commitmentIndexBytes, in.CommitmentIndex)
		nullifierPreimage := make([]byte, 0, 8+32+100)
		nullifierPreimage = append(nullifierPreimage, commitmentIndexBytes...)
		nullifierPreimage = append(nullifierPreimage, in.Salt[:]...)
		nullifierPreimage = append(nullifierPreimage, in.ScriptCommitment...)
		for _, param := range in.ScriptParams {
			nullifierPreimage = append(nullifierPreimage, param...)
		}
		calculatedNullifier := hash.HashFunc(nullifierPreimage)
		if !bytes.Equal(calculatedNullifier, pub.Nullifiers[i][:]) {
			return false
		}

		// Total up the input amounts
		if in.AssetID == defaultAssetID {
			inVal, err = AddUint64(inVal, in.Amount)
			if err != nil {
				return false
			}
		} else {
			assetIns[in.AssetID], err = AddUint64(assetIns[in.AssetID], in.Amount)
			if err != nil {
				return false
			}
		}
	}

	outVal := uint64(0)
	assetOuts := make(map[[types.AssetIDLen]byte]uint64)
	if len(priv.Outputs) != len(pub.Outputs) {
		return false
	}
	for i, out := range priv.Outputs {
		// Make sure the OutputCommitment provided in the PublicParams
		// actually matches the calculated output commitment. This prevents
		// someone from putting a different output hash containing a
		// different amount in the transactions.
		commitmentPreimage := make([]byte, 0, hash.HashSize+8+types.AssetIDLen+types.StateLen+types.SaltLen)
		amountBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(amountBytes, out.Amount)

		commitmentPreimage = append(commitmentPreimage, out.ScriptHash...)
		commitmentPreimage = append(commitmentPreimage, amountBytes...)
		commitmentPreimage = append(commitmentPreimage, out.AssetID[:]...)
		commitmentPreimage = append(commitmentPreimage, out.State[:]...)
		commitmentPreimage = append(commitmentPreimage, out.Salt[:]...)
		outputCommitment := hash.HashFunc(commitmentPreimage)

		if !bytes.Equal(outputCommitment, pub.Outputs[i].Commitment) {
			return false
		}

		if out.AssetID == defaultAssetID {
			outVal, err = AddUint64(outVal, out.Amount)
			if err != nil {
				return false
			}
		} else {
			assetOuts[out.AssetID], err = AddUint64(assetOuts[out.AssetID], out.Amount)
			if err != nil {
				return false
			}
		}
	}

	// Verify the transactions is not spending more than it is allowed to
	totalOut, err := AddUint64(outVal, pub.Fee)
	if err != nil {
		return false
	}
	totalIn, err := AddUint64(inVal, pub.Coinbase)
	if err != nil {
		return false
	}

	if totalOut > totalIn {
		return false
	}

	// Verify asset inputs and outputs add up.
	for assetID, outVal := range assetOuts {
		inVal := assetIns[assetID]
		if bytes.Equal(assetID[:], pub.MintID) {
			inVal, err = AddUint64(inVal, pub.MintAmount)
			if err != nil {
				return false
			}
		}
		if outVal > inVal {
			return false
		}
	}

	return true
}

func ValidateInclusionProof(outputCommitment []byte, commitmentIndex uint64, hashes [][]byte, flags uint64, accumulator [][]byte, root []byte) bool {
	// Prepend the output commitment wih the index and hash
	h := hash.HashWithIndex(outputCommitment, commitmentIndex)

	// Iterate over the hashes and hash with the previous has
	// using the flags to determine the ordering.
	for i := 0; i < len(hashes); i++ {
		eval := flags & (1 << i)
		if eval > 0 {
			h = hash.HashMerkleBranches(h, hashes[i])
		} else {
			h = hash.HashMerkleBranches(hashes[i], h)
		}
	}

	// Make sure the calculated hash exists in the accumulator (the root preimage)
	found := false
	for _, a := range accumulator {
		if bytes.Equal(a, h) {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// Hash the accumulator and compare to the root
	calculatedRoot := hash.CatAndHash(accumulator)
	return bytes.Equal(calculatedRoot, root)
}

// ValidateUnlockingScript is a placeholder. Normally this would be part of the overall circuit to validate
// the functional commitment.
func ValidateUnlockingScript(scriptCommitment []byte, scriptParams *UnlockingScriptInputs, unlockingParams [][]byte) (bool, error) {
	return true, nil
}

func AddUint64(a, b uint64) (uint64, error) {
	if b > 0 && a > math.MaxUint64-b {
		return 0, ErrIntegerOverflow
	}
	return a + b, nil
}
