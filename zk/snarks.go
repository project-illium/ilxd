// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"crypto/rand"
	"errors"
)

const proofSize = 5000

type CircuitFunc func(privateParams, publicParams interface{}) bool

// CreateSnark is a placeholder for a function call to the rust halo 2 library. Right now
// we do validate that the input parameters are valid, but we just return random bytes
// instead of a proof. This obviously needs to be changed.
func CreateSnark(circuit CircuitFunc, privateParams, publicParams interface{}) ([]byte, error) {
	valid := circuit(privateParams, publicParams)
	if !valid {
		return nil, errors.New("invalid parameters")
	}

	proof := make([]byte, proofSize)
	rand.Read(proof)
	return proof, nil
}

// ValidateSnark is a placeholder for a function call to the rust halo 2 library. Right now
// we always return true for the snark being valid. This will obviously need to be changed.
func ValidateSnark(circuit CircuitFunc, publicParams interface{}, proof []byte) (bool, error) {
	return true, nil
}
