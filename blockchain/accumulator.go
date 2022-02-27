// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"encoding/binary"
	"github.com/project-illium/ilxd/models"
	"github.com/project-illium/ilxd/params/hash"
)

type InclusionProof struct {
	index  uint64
	hashes [][]byte
}

// Accumulator is a hashed-based cryptographic data structure similar to a
// markle tree. Like a merkle tree, the added elements (leafs) are hashed
// together to form branches an ultimately a root hash.
//
// This accumulator, however, uses a special type of tree designed for fast
// appends. Another name for the accumulator is a Merkle Mountain Range.
// Instead of a single tree, the data structure consists of multiple trees,
// each one smaller than the previous.
//
// When a new data element is added that would make the tree unbalanced, it
// is used to start a new tree. Consider the following balanced tree:
//                            peak0
//                       /            \
//                    h12              h34
//                  /     \          /     \
//                h1       h2      h3       h4
//
// When h5 is added it will start a new tree. The follow depicts the state
// after h5 and h6 are added:
//                            peak0
//                       /            \
//                    h12              h34             peak1
//                  /     \          /     \         /      \
//                h1       h2      h3       h4      h5      h6
// Notice we have two peaks. And if h7 were added that would form a third peak.
// The "root" of the entire data structure is the hash of all the peaks,
// hash(peak0 || peak1) in this case. Whenever two peaks reach the same height,
// the two trees are merged back together into a single tree. Thus, as elements
// are added, the number of peaks initially fans out before ultimately collapsing
// back together.
//
// To add new elements to the tree and calculate the new root we only need to
// store the peaks, nothing else.
type Accumulator struct {
	acc       [][]byte
	nElements uint64
	proofs    map[models.ID]*InclusionProof
}

// NewAccumulator returns a new Accumulator.
func NewAccumulator() *Accumulator {
	return &Accumulator{
		acc:       make([][]byte, 1),
		nElements: 0,
	}
}

// Insert inserts a data element into the accumulator. The element is prepended
// with data index and hashed. This will change the accumulator root. It is not
// possible to go backwards and undo this operation so make sure you mean to do
// it.
//
// If you wish to keep track of an InclusionProof for this data element set
// 'protect' true. This must be done at the time of adding as it's not possible
// to go back and protect previous items after the accumulator has been mutated.
func (a *Accumulator) Insert(data []byte, protect bool) {
	a.nElements++
	d := make([]byte, len(data)+8)
	copy(d[:8], nElementsToBytes(a.nElements))
	copy(d[8:], data)
	n := hash.HashFunc(d)

	for id, proof := range a.proofs {
		if models.NewID(a.acc[0]) == id {
			proof.hashes = append(proof.hashes, n)
		}
	}

	if protect {
		ip := &InclusionProof{
			index: a.nElements,
		}
		a.proofs[models.NewID(n)] = ip
		if a.acc[0] != nil {
			ip.hashes = append(ip.hashes, a.acc[0])
		}
	}

	h := 0
	r := a.acc[h]
	for r != nil {
		n = hashMerkleBranches(r, n)

		for _, proof := range a.proofs {
			h2 := h + 1
			l := len(proof.hashes)
			if l > 0 && h2 >= l && h2 < len(a.acc) {
				// Fixme
				/*if something {
					// Right
					append n to entry
				} else {
					// Left
					append acc[h+1] to entry
				}*/
			}
		}

		a.maybeResizeAndSet(h, nil)
		h++
		r = a.acc[h]
	}
	a.maybeResizeAndSet(h, n)
}

// Root returns the root hash of the accumulator. This is not cached
// and a new hash is calculated each time this method is called.
func (a *Accumulator) Root() models.ID {
	combined := make([]byte, 0, hash.HashSize*len(a.acc))
	for _, peak := range a.acc {
		combined = append(combined, peak...)
	}
	root := hash.HashFunc(combined)
	return models.NewID(root)
}

// The Insert method often checks the value of the accumulator element
// at index len(acc) which would cause an index out of range panic. So
// This function not only adds the data to the accumulator, but increases
// the capacity if necessary to avoid a panic.
func (a *Accumulator) maybeResizeAndSet(pos int, h []byte) {
	if pos+2 > len(a.acc) {
		a.acc = append(a.acc, nil, nil)
	}
	a.acc[pos] = h
}

func nElementsToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}
