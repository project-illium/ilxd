// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import "github.com/project-illium/ilxd/repo"

type flushMode uint8

const (
	// flushRequired is used to signal that a validator set flush must take place.
	flushRequired flushMode = iota

	// flushPeriodic will flush if a certain time interval has passed since the last
	// flush.
	flushPeriodic

	// flushIfNeeded will flush if the cache goes over the max size in memeory.
	flushIfNeeded
)

type Blockchain struct {
	ds    repo.Datastore
	index *blockIndex
	vs    *ValidatorSet
	ns    *NullifierSet
}

func NewBlockchain(ds repo.Datastore) *Blockchain {
	return &Blockchain{}
}
