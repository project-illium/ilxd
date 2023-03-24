// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

// Amount represents the base illium monetary unit (to be named later).
// The total number of coins issued is not expected to overflow an uint64
// for 250 years from genesis.
type Amount uint64

// TODO: when the name of the base unit is decided on along with the number of
// base uints per ILX we will build out this class with functions that do coversion
// and formatting.
//
// For purposes of internal calculations, however we will always use base units.
