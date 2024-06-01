// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
)

// InclusionProof is a merkle inclusion proof which proves that
// a given element is in the set with the given accumulator root.
type InclusionProof struct {
	ID     types.ID
	Hashes [][]byte
	Flags  uint64
	Index  uint64
	last   []byte
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
//
//	            peak0
//	       /            \
//	    h12              h34
//	  /     \          /     \
//	h1       h2      h3       h4
//
// When h5 is added it will start a new tree. The follow depicts the state
// after h5 and h6 are added:
//
//	            peak0
//	       /            \
//	    h12              h34             peak1
//	  /     \          /     \         /      \
//	h1       h2      h3       h4      h5      h6
//
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
	proofs    map[types.ID]*InclusionProof
	lookupMap map[types.ID]uint64
}

// NewAccumulator returns a new Accumulator
func NewAccumulator() *Accumulator {
	return &Accumulator{
		acc:       make([][]byte, 1),
		proofs:    make(map[types.ID]*InclusionProof),
		lookupMap: make(map[types.ID]uint64),
		nElements: 0,
	}
}

// NewAccumulatorFromData returns a new accumulator from the raw data.
// The proof store will be initialized but empty.
func NewAccumulatorFromData(hashes [][]byte, nElements uint64) *Accumulator {
	if hashes == nil {
		hashes = [][]byte{}
	}

	return &Accumulator{
		acc:       hashes,
		nElements: nElements,
		proofs:    make(map[types.ID]*InclusionProof),
		lookupMap: make(map[types.ID]uint64),
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
	if data == nil {
		log.Warn("accumulator::Insert Attempted to insert nil data into the accumulator.")
		return
	}
	if len(data) == 0 {
		log.Warn("accumulator::Insert Attempted to insert empty data into the accumulator.")
		return
	}
	datacpy := make([]byte, len(data))
	copy(datacpy, data)
	n := hash.HashWithIndex(datacpy, a.nElements)
	a.nElements++

	// If one of our protected hashes is at acc[0] then it was an
	// odd number leaf and the very next leaf must be part of its
	// inclusion proof.
	proof, ok := a.proofs[types.NewID(a.acc[0])]
	if ok {
		c := make([]byte, len(n))
		copy(c, n)
		proof.Hashes = append(proof.Hashes, c)
		proof.last = hash.HashMerkleBranches(a.acc[0], n)
		proof.Flags = 1
	}

	if protect {
		ip := &InclusionProof{
			ID:    types.NewID(datacpy),
			Index: a.nElements - 1,
		}
		a.proofs[types.NewID(n)] = ip
		a.lookupMap[types.NewID(datacpy)] = ip.Index
		// If acc[0] is not nil then this means the new leaf is
		// and even number and the previous leaf is part of its
		// inclusion proof.
		if a.acc[0] != nil {
			c1 := make([]byte, len(a.acc[0]))
			c2 := make([]byte, len(n))
			copy(c1, a.acc[0])
			copy(c2, n)
			ip.Hashes = append(ip.Hashes, c1)
			ip.last = hash.HashMerkleBranches(a.acc[0], c2)
		}
	}
	accLen := a.len()
	h := 0
	r := a.acc[h]
	for r != nil {
		n = hash.HashMerkleBranches(r, n)

		// Iterate over all proofs and update them before we prune
		// branches off the tree.
		for _, proof := range a.proofs {
			h2 := h + 1
			l := len(proof.Hashes)
			if l > 0 && h2 >= l && h2 <= accLen {
				if !bytes.Equal(proof.last, n) { // Right
					c := make([]byte, len(n))
					copy(c, n)

					proof.Hashes = append(proof.Hashes, c)
					proof.last = hash.HashMerkleBranches(proof.last, n)

					f := uint64(1) << uint64(len(proof.Hashes)-1)
					proof.Flags |= f
				} else { // Left
					if len(a.acc)-1 < h+1 || a.acc[h+1] == nil {
						continue
					}
					c := make([]byte, len(a.acc[h+1]))
					copy(c, a.acc[h+1])

					proof.Hashes = append(proof.Hashes, c)
					proof.last = hash.HashMerkleBranches(a.acc[h+1], proof.last)
				}
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
func (a *Accumulator) Root() types.ID {
	if len(a.acc) == 0 {
		return types.ID{}
	}

	merkles := BuildMerkleTreeStore(reverseIDs(byteSliceToIDs(a.acc)))
	if len(merkles) == 0 {
		return types.ID{}
	}
	return types.NewID(merkles[len(merkles)-1])
}

// NumElements returns the current number of elements in the accumulator.
func (a *Accumulator) NumElements() uint64 {
	return a.nElements
}

// GetProof returns an inclusion proof, if it exists, for the provided hash.
//
// This is NOT safe for concurrent access.
func (a *Accumulator) GetProof(data []byte) (*InclusionProof, error) {
	idx, ok := a.lookupMap[types.NewID(data)]
	if !ok {
		return nil, errors.New("not found")
	}
	n := hash.HashWithIndex(data, idx)
	proof, ok := a.proofs[types.NewID(n)]
	if !ok {
		return nil, errors.New("not found")
	}

	for i := 0; i < len(proof.Hashes); i++ {
		eval := proof.Flags & (1 << i)
		if eval > 0 {
			n = hash.HashMerkleBranches(n, proof.Hashes[i])
		} else {
			n = hash.HashMerkleBranches(proof.Hashes[i], n)
		}
	}

	merkles := BuildMerkleTreeStore(reverseIDs(byteSliceToIDs(a.acc)))
	extraHashes, extraBits := MerkleInclusionProof(merkles, types.NewID(n))
	flags := proof.Flags | (uint64(extraBits) << len(proof.Hashes))

	newProof := &InclusionProof{
		ID:     types.NewID(data),
		Flags:  flags,
		Hashes: make([][]byte, len(proof.Hashes)+len(extraHashes)),
		Index:  proof.Index,
	}
	for i := range proof.Hashes {
		newProof.Hashes[i] = make([]byte, len(proof.Hashes[i]))
		copy(newProof.Hashes[i], proof.Hashes[i])
	}
	for i := range extraHashes {
		newProof.Hashes[i+len(proof.Hashes)] = make([]byte, len(extraHashes[i]))
		copy(newProof.Hashes[i+len(proof.Hashes)], extraHashes[i])
	}

	return newProof, nil
}

// DropProof ceases tracking of the inclusion proof for the given
// element and deletes all tree branches related to the proof.
//
// This is NOT safe for concurrent access.
func (a *Accumulator) DropProof(data []byte) {
	if data == nil {
		log.Warn("accumulator::DropProof Attempted to drop proof for nil data.")
		return
	}

	ixd, ok := a.lookupMap[types.NewID(data)]
	if !ok {
		return
	}

	n := hash.HashWithIndex(data, ixd)

	delete(a.lookupMap, types.NewID(data))
	delete(a.proofs, types.NewID(n))
}

// MergeProofs copes the inclusion proofs from the provided accumulator
// into this accumulator *only* if the proofs do not currently exist
// in this accumulator.
func (a *Accumulator) MergeProofs(acc *Accumulator) {
	for k, v := range acc.proofs {
		if _, ok := a.proofs[k]; ok {
			continue
		}
		cpy := &InclusionProof{
			ID:     v.ID.Clone(),
			Hashes: make([][]byte, len(v.Hashes)),
			Flags:  v.Flags,
			Index:  v.Index,
			last:   make([]byte, len(v.last)),
		}
		for i := range v.Hashes {
			cpy.Hashes[i] = make([]byte, len(v.Hashes[i]))
			copy(cpy.Hashes[i], v.Hashes[i])
		}
		copy(cpy.last, v.last)
		a.proofs[k] = cpy
	}
	for k, v := range acc.lookupMap {
		if _, ok := a.lookupMap[k]; ok {
			continue
		}
		a.lookupMap[k.Clone()] = v
	}
}

// Hashes returns the accumulator hashes
func (a *Accumulator) Hashes() [][]byte {
	return a.acc
}

// Clone returns a copy of the accumulator. Modifications to the copy will not
// affect the original.
func (a *Accumulator) Clone() *Accumulator {
	acc := make([][]byte, len(a.acc), cap(a.acc))
	for x := range a.acc {
		acc[x] = make([]byte, len(a.acc[x]))
		if a.acc[x] == nil {
			acc[x] = nil
		} else {
			copy(acc[x], a.acc[x])
		}
	}

	proofs := make(map[types.ID]*InclusionProof)
	for key, proof := range a.proofs {
		i := InclusionProof{
			ID:     proof.ID,
			Flags:  proof.Flags,
			Index:  proof.Index,
			Hashes: make([][]byte, len(proof.Hashes)),
			last:   make([]byte, len(proof.last)),
		}
		for x := range proof.Hashes {
			i.Hashes[x] = make([]byte, len(proof.Hashes[x]))
			copy(i.Hashes[x], proof.Hashes[x])
		}
		copy(i.last, proof.last)
		proofs[key] = &i
	}
	lookupMap := make(map[types.ID]uint64)
	for key, idx := range a.lookupMap {
		k := make([]byte, len(key.Bytes()))
		copy(k, key.Bytes())
		lookupMap[types.NewID(k)] = idx
	}

	return &Accumulator{
		acc:       acc,
		nElements: a.nElements,
		proofs:    proofs,
		lookupMap: lookupMap,
	}
}

// ValidateInclusionProof validates that the inclusion proof links the data
// and commitment index to the merkle root. This function mirrors the functionality
// inside the transaction validation lurk programs.
func ValidateInclusionProof(data []byte, commitmentIndex uint64, hashes [][]byte, flags uint64, root []byte) (bool, error) {
	// Prepend the output commitment wih the index and hash
	h := hash.HashWithIndex(data, commitmentIndex)

	// Iterate over the hashes and hash with the previous has
	// using the flags to determine the ordering.
	for i := 0; i < len(hashes); i++ {
		eval := flags & (1 << i)
		if eval > 0 {
			h = hash.HashMerkleBranches(h, hashes[i])
		} else {
			h = hash.HashMerkleBranches(hashes[i], h)
		}
	}

	return bytes.Equal(h, root), nil
}

// EvalInclusionProof also validates the inclusion proof but uses the lurk program
// that is used to validate transactions to do so.
func EvalInclusionProof(data []byte, commitmentIndex uint64, hashes [][]byte, flags uint64, root []byte) (bool, error) {
	h := hash.HashWithIndex(data, commitmentIndex)

	ip := &circparams.InclusionProof{
		Hashes: hashes,
		Flags:  flags,
	}

	ipExpr, err := ip.ToExpr()
	if err != nil {
		return false, err
	}

	program := fmt.Sprintf(`
			(lambda (priv pub)
				(letrec (
                            (cat-and-hash (lambda (a b)
                                    			(eval (cons 'coproc_blake2s (cons a (cons b nil))))))

							(validate-inclusion-proof (lambda (leaf hashes root)
								   (letrec (
									   (hash-branches (lambda (h hashes)
										   (let ((next-hash (car hashes))
												(val (car next-hash))
												(new-h (if (cdr next-hash)
													  (cat-and-hash h val)
													  (cat-and-hash val h))))
				
											  (if (cdr hashes)
												  (hash-branches new-h (cdr hashes))
												  new-h)))))
				
									   (if hashes
										   (= (hash-branches leaf hashes) root) ;; All others
										   (= leaf root)                        ;; Genesis coinbase
									   ))))
							)
							(validate-inclusion-proof 0x%x %s 0x%x)
				)
			)
	`, h, ipExpr, root)

	tag, output, _, err := zk.Eval(program, zk.Expr("nil"), zk.Expr("nil"))
	if err != nil {
		return false, err
	}
	return tag == zk.TagSym && bytes.Equal(output, zk.OutputTrue), nil
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

func (a *Accumulator) len() int {
	l := 0
	for _, d := range a.acc {
		if d != nil {
			l++
		}
	}
	return l
}

func reverseIDs(s []types.ID) []types.ID {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
