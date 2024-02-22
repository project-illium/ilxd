// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"bytes"
	"crypto/rand"
	"errors"
	"sync"
)

// Parameters is an interface for script private or public
// parameters that converts a struct to a lurk list usable
// by a script.
type Parameters interface {
	// ToExpr marshals the Parameters to a string
	// expression used by lurk.
	ToExpr() (string, error)
}

// Prover is an interface to the zk-snark prove function.
type Prover interface {
	// Prove creates a proof that the private and public params
	// make the program return true.
	//
	// maxSteps is optional if you want to limit the number of steps
	// before terminating the proof creation.
	Prove(program string, privateParams Parameters, publicParams Parameters, maxSteps ...uint64) (proof []byte, err error)
}

// Verifier is an interface to the zk-snark verify function.
type Verifier interface {
	// Verify uses the public params and the proof to verify that
	// the program returned true.
	Verify(program string, publicParams Parameters, proof []byte) (valid bool, err error)
}

// LurkProver is an implementation of the Prover interface
// that creates lurk zk-snark proofs.
type LurkProver struct{}

// Prove creates a proof that the private and public params
// make the program return true.
func (l *LurkProver) Prove(program string, privateParams Parameters, publicParams Parameters, maxSteps ...uint64) (proof []byte, err error) {
	return Prove(program, privateParams, publicParams, maxSteps...)
}

// LurkVerifier is an implementation of the Verifier interface
// that verifies lurk zk-snark proofs.
type LurkVerifier struct{}

// Verify uses the public params and the proof to verify that
// the program returned true.
func (l *LurkVerifier) Verify(program string, publicParams Parameters, proof []byte) (valid bool, err error) {
	return Verify(program, publicParams, proof)
}

// MockProver is a mock implementation of the Prover interface.
// It does validate that the private and public parameters make
// the program return true, but it does not actually create the
// proof. Instead, it just returns random bytes.
type MockProver struct {
	proofLen int
	mtx      sync.RWMutex
}

// Prove creates a proof that the private and public params
// make the program return true.
func (m *MockProver) Prove(program string, privateParams Parameters, publicParams Parameters, maxSteps ...uint64) ([]byte, error) {
	priv, err := privateParams.ToExpr()
	if err != nil {
		return nil, err
	}
	pub, err := publicParams.ToExpr()
	if err != nil {
		return nil, err
	}

	ms := defaultMaxSteps
	if len(maxSteps) > 0 {
		ms = maxSteps[0]
	}

	tag, val, _, err := evaluate(program, priv, pub, ms, false)
	if err != nil {
		return nil, err
	}
	if tag != TagSym {
		return nil, errors.New("program did not return true")
	}
	if !bytes.Equal(val, OutputTrue) {
		return nil, errors.New("program did not return true")
	}
	proofLen := EstimatedProofSize
	m.mtx.RLock()
	if m.proofLen > 0 {
		proofLen = m.proofLen
	}
	m.mtx.RUnlock()
	proof := make([]byte, proofLen)
	rand.Read(proof)
	return proof, nil
}

// SetProofLen sets the length of the mock proof returned
// by the Prove method.
func (m *MockProver) SetProofLen(length int) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.proofLen = length
}

// MockVerifier does nto validate the proof at all and just
// returns the value of valid instead.
type MockVerifier struct {
	valid bool
	mtx   sync.RWMutex
}

// Verify uses the public params and the proof to verify that
// the program returned true.
func (m *MockVerifier) Verify(program string, publicParams Parameters, proof []byte) (valid bool, err error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.valid, nil
}

// SetValid sets the return value for the Verify method
func (m *MockVerifier) SetValid(valid bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.valid = valid
}
