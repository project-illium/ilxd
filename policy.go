// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import "github.com/project-illium/ilxd/types"

type Policy struct {
	MinFeePerByte      types.Amount
	MinStake           types.Amount
	BlocksizeSoftLimit uint32
	MaxMessageSize     uint32
	TreasuryWhitelist  []types.ID
}
