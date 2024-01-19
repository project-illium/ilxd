// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/types"
	"sync"
)

type sigCacheEntry struct {
	sig    []byte
	pubKey crypto.PubKey
}

// SigCache is used to cache the validation of transaction signatures.
// Transactions are typically validated twice. Once when they enter the
// mempool and once again when a block is connected to the chain. We
// cache the validated signatures here to avoid having to redo expensive
// computation.
type SigCache struct {
	sync.RWMutex
	validSigs  map[types.ID]sigCacheEntry
	maxEntries uint
}

// NewSigCache returns an instantiated SigCache. maxEntries can be used
// to control memory usage.
func NewSigCache(maxEntries uint) *SigCache {
	return &SigCache{
		validSigs:  make(map[types.ID]sigCacheEntry, maxEntries),
		maxEntries: maxEntries,
	}
}

// Exists returns whether the signature exists in the cache.
func (s *SigCache) Exists(sigHash types.ID, sig []byte, pubKey crypto.PubKey) bool {
	s.RLock()
	entry, ok := s.validSigs[sigHash]
	s.RUnlock()

	return ok && entry.pubKey.Equals(pubKey) && bytes.Equal(entry.sig, sig)
}

// Add will add a new signature to the cache. If the new signature would exceed maxEntries
// a random signature will be evicted from the cache.
//
// NOTE: Signatures should be validated before adding to this cache and only valid
// signatures should ever be added.
func (s *SigCache) Add(sigHash types.ID, sig []byte, pubKey crypto.PubKey) {
	s.Lock()
	defer s.Unlock()

	if s.maxEntries <= 0 {
		return
	}

	// If adding this new entry will put us over the max number of allowed
	// entries, then evict an entry.
	if uint(len(s.validSigs)+1) > s.maxEntries {
		// Remove a random entry from the map. Relying on the random
		// starting point of Go's map iteration. It's worth noting that
		// the random iteration starting point is not 100% guaranteed
		// by the spec, however most Go compilers support it.
		// Ultimately, the iteration order isn't important here because
		// in order to manipulate which items are evicted, an adversary
		// would need to be able to execute preimage attacks on the
		// hashing function in order to start eviction at a specific
		// entry.
		for sigEntry := range s.validSigs {
			delete(s.validSigs, sigEntry)
			break
		}
	}
	s.validSigs[sigHash] = sigCacheEntry{sig, pubKey}
}
