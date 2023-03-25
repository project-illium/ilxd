// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"github.com/project-illium/ilxd/types/transactions"
)

// TxSorter implements sort.Interface to allow a slice of transactions
// to be sorted lexicographically.
type TxSorter []*transactions.Transaction

// Len returns the number of txs in the slice.  It is part of the
// sort.Interface implementation.
func (s TxSorter) Len() int {
	return len(s)
}

// Swap swaps the txs at the passed indices.  It is part of the
// sort.Interface implementation.
func (s TxSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the txs with index i should sort before the
// tx with index j.  It is part of the sort.Interface implementation.
func (s TxSorter) Less(i, j int) bool {
	return s[i].ID().Compare(s[j].ID()) < 0
}
