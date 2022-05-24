// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"github.com/project-illium/ilxd/types"
	"sync"
)

type proofCacheEntry struct {
	proof []byte
}

type ProofCache struct {
	sync.RWMutex
	validProofs map[types.ID]proofCacheEntry
	maxEntries  uint
}

func NewProofCache(maxEntries uint) *ProofCache {
	return &ProofCache{
		validProofs: make(map[types.ID]proofCacheEntry, maxEntries),
		maxEntries:  maxEntries,
	}
}

func (p *ProofCache) Exists(proofHash types.ID, proof []byte) bool {
	p.RLock()
	entry, ok := p.validProofs[proofHash]
	p.RUnlock()

	return ok && bytes.Equal(entry.proof, proof)
}

func (p *ProofCache) Add(proofHash types.ID, proof []byte) {
	p.Lock()
	defer p.Unlock()

	if p.maxEntries <= 0 {
		return
	}

	// If adding this new entry will put us over the max number of allowed
	// entries, then evict an entry.
	if uint(len(p.validProofs)+1) > p.maxEntries {
		// Remove a random entry from the map. Relying on the random
		// starting point of Go's map iteration. It's worth noting that
		// the random iteration starting point is not 100% guaranteed
		// by the spec, however most Go compilers support it.
		// Ultimately, the iteration order isn't important here because
		// in order to manipulate which items are evicted, an adversary
		// would need to be able to execute preimage attacks on the
		// hashing function in order to start eviction at a specific
		// entry.
		for proofEntry := range p.validProofs {
			delete(p.validProofs, proofEntry)
			break
		}
	}
	p.validProofs[proofHash] = proofCacheEntry{proof}
}
