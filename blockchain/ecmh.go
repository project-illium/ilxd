// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/project-illium/ilxd/models"
	"golang.org/x/crypto/blake2b"
	"math/big"
	"sync"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// Multiset tracks the state of a multiset as used to calculate the ECMH
// (elliptic curve multiset hash) hash of an unordered set. The state is
// a point on the curve. New elements are hashed onto a point on the curve
// and then added to the current state. Hence elements can be added in any
// order and we can also remove elements to return to a prior hash.
type Multiset struct {
	curve *secp.KoblitzCurve
	point *secp.JacobianPoint
	mtx   sync.RWMutex
}

// NewMultiset returns an empty multiset. The hash of an empty set
// is the 32 byte value of zero.
func NewMultiset(curve *secp.KoblitzCurve) *Multiset {
	return &Multiset{curve: curve, point: new(secp.JacobianPoint), mtx: sync.RWMutex{}}
}

// NewMultisetFromPoint initializes a new multiset with the given x, y
// coordinate.
func NewMultisetFromPoint(curve *secp.KoblitzCurve, x, y *big.Int) *Multiset {
	var copyX, copyY big.Int
	if x != nil {
		copyX.Set(x)
	}
	if y != nil {
		copyY.Set(y)
	}
	result := new(secp.JacobianPoint)
	bigAffineToJacobian(x, y, result)

	return &Multiset{curve: curve, point: result, mtx: sync.RWMutex{}}
}

// Add hashes the data onto the curve and updates the state
// of the multiset.
func (ms *Multiset) Add(data []byte) {
	ms.mtx.Lock()
	defer ms.mtx.Unlock()

	x, y := hashToPoint(ms.curve, data)

	point2 := &secp.JacobianPoint{}
	point2.X.SetByteSlice(x.Bytes())
	point2.Y.SetByteSlice(y.Bytes())
	point2.Z.SetInt(1)

	result := new(secp.JacobianPoint)

	secp.AddNonConst(ms.point, point2, result)

	ms.point = result
}

// Remove hashes the data onto the curve and subtracts the value
// from the state. This function will execute regardless of whether
// or not the passed data was previously added to the set. Hence if
// you remove and element that was never added and also remove all the
// elements that were added, you will not get back to the point at
// infinity (empty set).
func (ms *Multiset) Remove(data []byte) {
	ms.mtx.Lock()
	defer ms.mtx.Unlock()

	if ms.point.X.IsZero() && ms.point.Y.IsZero() {
		return
	}

	x, y := hashToPoint(ms.curve, data)
	y = y.Neg(y).Mod(y, ms.curve.P)

	point2 := &secp.JacobianPoint{}
	bigAffineToJacobian(x, y, point2)

	result := new(secp.JacobianPoint)
	secp.AddNonConst(ms.point, point2, result)

	ms.point = result
}

// Hash serializes and returns the hash of the multiset. The hash of an empty
// set is the 32 byte value of zero. The hash of a non-empty multiset is the
// sha256 hash of the 32 byte x value concatenated with the 32 byte y value.
func (ms *Multiset) Hash() models.ID {
	ms.mtx.RLock()
	defer ms.mtx.RUnlock()

	if ms.point.X.IsZero() && ms.point.Y.IsZero() {
		return models.ID{}
	}

	cpy := *ms.point
	cpy.ToAffine()

	// Convert the field values for the now affine point to big.Ints.
	x3, y3 := new(big.Int), new(big.Int)
	x3.SetBytes(cpy.X.Bytes()[:])
	y3.SetBytes(cpy.Y.Bytes()[:])

	return models.NewIDFromData(append(x3.Bytes(), y3.Bytes()...))
}

// Point returns a copy of the x and y coordinates of the current multiset state.
func (ms *Multiset) Point() (x *big.Int, y *big.Int) {
	ms.mtx.RLock()
	defer ms.mtx.RUnlock()

	return jacobianToBigAffine(ms.point)
}

// hashToPoint hashes the passed data into a point on the curve. The x value
// is sha256(n, sha256(data)) where n starts at zero. If the resulting x value
// is not in the field or x^3+7 is not quadratic residue then n is incremented
// and we try again. There is a 50% chance of success for any given iteration.
func hashToPoint(curve *secp.KoblitzCurve, data []byte) (*big.Int, *big.Int) {
	var (
		i    = uint64(0)
		x, y *secp.FieldVal
		bigX *big.Int
		h    = blake2b.Sum256(data)
		n    = make([]byte, 8)
	)

	for {
		binary.BigEndian.PutUint64(n, i)
		h2 := sha256.Sum256(append(n, h[:]...))

		x, y = new(secp.FieldVal), new(secp.FieldVal)
		x.SetBytes(&h2)

		bigX = new(big.Int).SetBytes(h2[:])

		yes := secp.DecompressY(x, false, y)
		if yes && bigX.Cmp(curve.N) < 0 {
			y.Normalize()
			break
		}
		i++
	}
	return bigX, new(big.Int).SetBytes(y.Bytes()[:])
}

func jacobianToBigAffine(point *secp.JacobianPoint) (*big.Int, *big.Int) {
	point.ToAffine()

	// Convert the field values for the now affine point to big.Ints.
	x3, y3 := new(big.Int), new(big.Int)
	x3.SetBytes(point.X.Bytes()[:])
	y3.SetBytes(point.Y.Bytes()[:])
	return x3, y3
}

func bigAffineToJacobian(x, y *big.Int, result *secp.JacobianPoint) {
	result.X.SetByteSlice(x.Bytes())
	result.Y.SetByteSlice(y.Bytes())
	result.Z.SetInt(1)
}
