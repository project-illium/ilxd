// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import "bytes"

type PrivateParams struct {
	Inputs []struct {
		SpendScript     []byte
		Amount          uint64
		Salt            []byte
		AssetID         [32]byte
		CommitmentIndex int
		MerkleProof     []byte
		Threshold       int
		Pubkeys         [][]byte
		Signatures      [][]byte
		SigBitfield     byte
	}
	Outputs []struct {
		SpendScript []byte
		Amount      uint64
		Salt        []byte
		AssetID     [32]byte
	}
}

type PublicParams struct {
	UTXORoot          []byte
	SigHash           []byte
	OutputCommitments [][]byte
	Nullifiers        [][]byte
	Fee               uint64
	Coinbase          uint64
	MintID            []byte
	MintAmount        uint64
}

// Placeholder, FIXME
func StandardCircuit(priv PrivateParams, pub PublicParams) bool {
	inVal := uint64(0)
	assetIns := make(map[[32]byte]uint64)

	for i, in := range priv.Inputs {
		// First obtain the hash of the UTXO
		outputCommitment := Hash(Serialize(in.SpendScript, in.Amount, in.Salt, in.AssetID))

		// Then validate the merkle proof
		if !ValidateMerkleProof(outputCommitment, in.CommitmentIndex, in.MerkleProof, pub.UTXORoot) {
			return false
		}

		// Validate that the provided threshold and pubkeys hash to the keyHash
		calculatedSpendScript := CalculateSpendScript(in.Threshold, in.Pubkeys)
		if calculatedSpendScript != in.SpendScript {
			return false
		}

		// Validate the signature(s)
		if !ValidateMultiSignature(in.Threshold, in.Pubkeys, in.Signatures, in.SigBitfield, pub.SigHash) {
			return false
		}

		// Validate that the nullifier is calculated correctly.
		calculatedNullifier := Hash(in.CommitmentIndex, in.Threshold, in.Salt, in.Pubkeys...)
		if !bytes.Equal(calculatedNullifier, pub.Nullifiers[i]) {
			return false
		}

		// Total up the input amounts
		if in.AssetID == DefaultAssetID {
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
		outputCommitment := Hash(Serialize(out.SpendScript, out.Amount, out.Salt))
		if !bytes.Equal(outputCommitment, pub.OutputCommitments[i]) {
			return false
		}

		if out.AssetID == DefaultAssetID {
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
		if bytes.Equal(assetID, pub.MintID) {
			inVal += pub.MintAmount
		}
		if outVal > inVal {
			return false
		}
	}

	return true
}
