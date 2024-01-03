// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import "github.com/project-illium/ilxd/types"

type PrivateInput struct {
	ScriptHash       types.ID
	Amount           types.Amount
	AssetID          types.ID
	State            types.State
	Salt             types.ID
	CommitmentIndex  uint64
	InclusionProof   InclusionProof
	ScriptCommitment types.ID
	ScriptParams     [][]byte
	UnlockingParams  string
}

type PrivateOutput struct {
	ScriptHash types.ID
	Amount     types.Amount
	AssetID    types.ID
	State      types.State
	Salt       types.ID
}

type PublicOutput struct {
	Commitment types.ID
	CipherText []byte
}

type InclusionProof struct {
	Hashes [][]byte
	Flags  uint64
}
