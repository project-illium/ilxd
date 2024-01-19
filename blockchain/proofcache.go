// Copyright (c) 2024 The illium developers
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
	txid  types.ID
}

// ProofCache is used to cache the validation of zero knowledge proofs.
// Transactions are typically validated twice. Once when they enter the
// mempool and once again when a block is connected to the chain. We
// cache the validated proofs here to avoid having to redo expensive
// computation.
type ProofCache struct {
	sync.RWMutex
	validProofs map[types.ID]proofCacheEntry
	maxEntries  uint
}

// NewProofCache returns an instantiated ProofCache. maxEntries can be used
// to control memory usage.
func NewProofCache(maxEntries uint) *ProofCache {
	return &ProofCache{
		validProofs: make(map[types.ID]proofCacheEntry, maxEntries),
		maxEntries:  maxEntries,
	}
}

// Exists returns whether the proof exists in the cache.
func (p *ProofCache) Exists(proofHash types.ID, proof []byte, txid types.ID) bool {
	p.RLock()
	entry, ok := p.validProofs[proofHash]
	p.RUnlock()

	return ok && entry.txid == txid && bytes.Equal(entry.proof, proof)
}

// Add will add a new proof to the cache. If the new proof would exceed maxEntries
// a random proof will be evicted from the cache.
//
// NOTE: Proofs should be validated before adding to this cache and only valid
// proofs should ever be added.
func (p *ProofCache) Add(proofHash types.ID, proof []byte, txid types.ID) {
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
	p.validProofs[proofHash] = proofCacheEntry{proof, txid}
}
