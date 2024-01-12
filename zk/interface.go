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

// Prover is an interface to the zk-snark prove function.
type Prover interface {
	// Prove creates a proof that the private and public params
	// make the program return true.
	Prove(program string, privateParams Parameters, publicParams Parameters) (proof []byte, err error)
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
func (l *LurkProver) Prove(program string, privateParams Parameters, publicParams Parameters) (proof []byte, err error) {
	return Prove(program, privateParams, publicParams)
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
type MockProver struct{}

// Prove creates a proof that the private and public params
// make the program return true.
func (m *MockProver) Prove(program string, privateParams Parameters, publicParams Parameters) ([]byte, error) {
	tag, val, _, err := Eval(program, privateParams, publicParams)
	if err != nil {
		return nil, err
	}
	if tag != TagSym {
		return nil, errors.New("program did not return true")
	}
	if !bytes.Equal(val, OutputTrue) {
		return nil, errors.New("program did not return true")
	}
	proof := make([]byte, 11000)
	rand.Read(proof)
	return proof, nil
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
