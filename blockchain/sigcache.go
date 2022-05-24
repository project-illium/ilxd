// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/types"
	"sync"
)

type sigCacheEntry struct {
	sig    []byte
	pubKey crypto.PubKey
}

type SigCache struct {
	sync.RWMutex
	validSigs  map[types.ID]sigCacheEntry
	maxEntries uint
}

func NewSigCache(maxEntries uint) *SigCache {
	return &SigCache{
		validSigs:  make(map[types.ID]sigCacheEntry, maxEntries),
		maxEntries: maxEntries,
	}
}

func (s *SigCache) Exists(sigHash types.ID, sig []byte, pubKey crypto.PubKey) bool {
	s.RLock()
	entry, ok := s.validSigs[sigHash]
	s.RUnlock()

	return ok && entry.pubKey.Equals(pubKey) && bytes.Equal(entry.sig, sig)
}

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
