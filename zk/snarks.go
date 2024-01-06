// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const MockProofSize = 9000

const LurkMaxFieldElement = "40000000000000000000000000000000224698fc0994a8dd8c46eb2100000000"

type CircuitFunc func(privateParams, publicParams interface{}) bool

// CreateSnark is a placeholder for a function call to the rust lurk library. Right now
// we do validate that the input parameters are valid, but we just return random bytes
// instead of a proof. This obviously needs to be changed.
func CreateSnark(circuit CircuitFunc, privateParams, publicParams interface{}) ([]byte, error) {
	valid := circuit(privateParams, publicParams)
	if !valid {
		return nil, errors.New("invalid parameters")
	}

	proof := make([]byte, MockProofSize)
	rand.Read(proof)
	return proof, nil
}

// ValidateSnark is a placeholder for a function call to the rust lurk library. Right now
// we always return true for the snark being valid. This will obviously need to be changed.
func ValidateSnark(circuit CircuitFunc, publicParams interface{}, proof []byte) (bool, error) {
	return true, nil
}

// SignatureToExpression converts a 64 byte signature to a lurk cons expression
// containing the signature's R and S values.
func SignatureToExpression(sig []byte) string {
	if len(sig) != 64 {
		return ""
	}
	sigR := sig[:32]
	sigS := sig[32:]

	bigR := new(big.Int).SetBytes(sigR)
	bigS := new(big.Int).SetBytes(sigS)

	return fmt.Sprintf("(cons %s %s)", bigR.String(), bigS.String())
}
