// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package standard

import (
	"bytes"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/params/hash"
	"math/bits"
	"time"
)

var defaultAssetID [32]byte

type InclusionProof struct {
	Hashes      [][]byte
	Flags       uint64
	Accumulator [][]byte
}

type PrivateInput struct {
	Amount          uint64
	Salt            []byte
	AssetID         [32]byte
	CommitmentIndex uint64
	InclusionProof  InclusionProof
	Threshold       uint8
	Pubkeys         [][]byte
	Signatures      [][]byte
	SigBitfield     uint8
}

type PrivateOutput struct {
	SpendScript []byte
	Amount      uint64
	Salt        []byte
	AssetID     [32]byte
}

type PrivateParams struct {
	Inputs  []PrivateInput
	Outputs []PrivateOutput
}

type PublicParams struct {
	TXORoot           []byte
	SigHash           []byte
	OutputCommitments [][]byte
	Nullifiers        [][32]byte
	Fee               uint64
	Coinbase          uint64
	MintID            []byte
	MintAmount        uint64
	Blocktime         time.Time
}

// This whole function is a placeholder for the actual zk-snark circuit. We enumerate it
// here to give an approximate idea of what the circuit will do.
func StandardCircuit(priv PrivateParams, pub PublicParams) bool {
	inVal := uint64(0)
	assetIns := make(map[[32]byte]uint64)

	for i, in := range priv.Inputs {
		// First obtain the hash of the spendScript.
		spendScriptPreimage := []byte{in.Threshold}
		for _, key := range in.Pubkeys {
			spendScriptPreimage = append(spendScriptPreimage, key...)
		}
		spendScriptHash := hash.HashFunc(spendScriptPreimage)

		amountBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(amountBytes, in.Amount)
		commitmentPreimage := make([]byte, 0, 32+8+32+32)
		commitmentPreimage = append(commitmentPreimage, spendScriptHash...)
		commitmentPreimage = append(commitmentPreimage, amountBytes...)
		commitmentPreimage = append(commitmentPreimage, in.AssetID[:]...)
		commitmentPreimage = append(commitmentPreimage, in.Salt...)
		outputCommitment := hash.HashFunc(commitmentPreimage)

		// Then validate the merkle proof
		if !ValidateInclusionProof(outputCommitment, in.CommitmentIndex, in.InclusionProof.Hashes, in.InclusionProof.Flags, in.InclusionProof.Accumulator, pub.TXORoot) {
			return false
		}

		// Validate the signature(s)
		valid, err := ValidateMultiSignature(in.Threshold, in.Pubkeys, in.Signatures, in.SigBitfield, pub.SigHash, pub.Blocktime)
		if !valid || err != nil {
			return false
		}

		// Validate that the nullifier is calculated correctly.
		commitmentIndexBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(commitmentIndexBytes, in.CommitmentIndex)
		nullifierPreimage := make([]byte, 0, 8+32+1+(len(in.Pubkeys)*32))
		nullifierPreimage = append(nullifierPreimage, commitmentIndexBytes...)
		nullifierPreimage = append(nullifierPreimage, in.Salt...)
		nullifierPreimage = append(nullifierPreimage, in.Threshold)
		for _, key := range in.Pubkeys {
			nullifierPreimage = append(nullifierPreimage, key...)
		}
		calculatedNullifier := hash.HashFunc(nullifierPreimage)
		if !bytes.Equal(calculatedNullifier, pub.Nullifiers[i][:]) {
			return false
		}

		// Total up the input amounts
		if in.AssetID == defaultAssetID {
			inVal += in.Amount
		} else {
			assetIns[in.AssetID] += in.Amount
		}
	}

	outVal := uint64(0)
	assetOuts := make(map[[32]byte]uint64)
	for i, out := range priv.Outputs {
		// Make sure the OutputCommitment provided in the PublicParams
		// actually matches the calculated output commitment. This prevents
		// someone from putting a different output hash containing a
		// different amount in the transactions.
		commitmentPreimage := make([]byte, 0, 32+8+32+32)
		amountBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(amountBytes, out.Amount)
		commitmentPreimage = append(commitmentPreimage, out.SpendScript...)
		commitmentPreimage = append(commitmentPreimage, amountBytes...)
		commitmentPreimage = append(commitmentPreimage, out.AssetID[:]...)
		commitmentPreimage = append(commitmentPreimage, out.Salt...)
		outputCommitment := hash.HashFunc(commitmentPreimage)
		if !bytes.Equal(outputCommitment, pub.OutputCommitments[i]) {
			return false
		}

		if out.AssetID == defaultAssetID {
			outVal += out.Amount
		} else {
			assetOuts[out.AssetID] += out.Amount
		}
	}

	// Verify the transactions is not spending more than it is allowed to
	if outVal+pub.Fee > inVal+pub.Coinbase {
		return false
	}

	// Verify asset inputs and outputs add up.
	for assetID, outVal := range assetOuts {
		inVal := assetIns[assetID]
		if bytes.Equal(assetID[:], pub.MintID) {
			inVal += pub.MintAmount
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

func ValidateMultiSignature(threshold uint8, pubkeys [][]byte, signatures [][]byte, sigBitField uint8, sigHash []byte, blockTime time.Time) (bool, error) {
	if len(signatures) > 8 || uint8(len(signatures)) < threshold {
		return false, nil
	}
	if bits.OnesCount8(sigBitField) != len(signatures) {
		return false, nil
	}
	if len(pubkeys) > 8 {
		return false, nil
	}
	sigIndex := 0
	for i := 0; i < len(pubkeys); i++ {
		f := uint8(1 << i)
		if f&sigBitField > 1 {
			timeLock := binary.BigEndian.Uint64(pubkeys[i][32:])
			if time.Unix(int64(timeLock), 0).After(blockTime) {
				return false, nil
			}

			pubkey, err := crypto.UnmarshalEd25519PublicKey(pubkeys[i][:32])
			if err != nil {
				return false, err
			}

			valid, err := pubkey.Verify(sigHash, signatures[sigIndex])
			if !valid || err != nil {
				return false, err
			}

			sigIndex++
		}
	}

	return true, nil
}
