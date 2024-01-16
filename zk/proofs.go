//go:build !skiprusttests
// +build !skiprusttests

// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

/*
#cgo linux CFLAGS: -Irust/target/release -Irust-target-release-zk
#cgo linux LDFLAGS: -Lrust/target/release -Lrust-target-release-zk -l:libillium_zk.a -ldl -lpthread -lgcc_s -lc -lm -lssl -lcrypto -lstdc++ -L/usr/local/cuda/lib64 -L/usr/local/cuda/targets/x86_64-linux/lib -lcudart
#cgo windows CFLAGS: -Irust/target/release
#cgo windows LDFLAGS: -Lrust/target/release -l:libillium_zk.lib
#cgo darwin CFLAGS: -Irust/target/release -Irust/target/x86_64-apple-darwin/release
#cgo darwin LDFLAGS: -Lrust/target/release -Lrust/target/x86_64-apple-darwin/release -lillium_zk -lc++ -lssl -lcrypto -framework SystemConfiguration
#include <stdlib.h>
#include <stdint.h>
void load_public_params();
int create_proof_ffi(
    const char* lurk_program,
    const char* private_params,
    const char* public_params,
    uint8_t* proof,
    size_t* proof_len,
    uint8_t* output_tag,
    uint8_t* output_val);
int verify_proof_ffi(
    const char* lurk_program,
    const char* public_params,
    const uint8_t* proof,
    size_t proof_size,
    const uint8_t* expected_tag,
    const uint8_t* expected_output);
int eval_ffi(
    const char* lurk_program,
    const char* private_params,
    const char* public_params,
    uint8_t* output_tag,
    uint8_t* output_val,
	size_t* iterations);
*/
import "C"
import (
	"bytes"
	"errors"
	"sync"
	"unsafe"
)

const (
	// EstimatedProofSize is the estimated size (in bytes) of the transaction
	// proofs. These vary slightly for each transaction type.
	EstimatedProofSize = 11500

	// LurkMaxFieldElement is the maximum value for a field element in lurk.
	// In practice this means lurk script variables cannot exceed this value.
	LurkMaxFieldElement = "40000000000000000000000000000000224698fc0994a8dd8c46eb2100000000"
)

var once sync.Once

// Parameters is an interface for script private or public
// parameters that converts a struct to a lurk list usable
// by a script.
type Parameters interface {
	// ToExpr marshals the Parameters to a string
	// expression used by lurk.
	ToExpr() (string, error)
}

// Expr is a Parameters type that wraps a string expression
type Expr string

func (p Expr) ToExpr() (string, error) {
	return string(p), nil
}

// LoadZKPublicParameters loads the lurk public parameters from disk
// into memory or generates them if this is the first startup.
func LoadZKPublicParameters() {
	once.Do(func() {
		log.Info("Loading lurk public parameters...")
		C.load_public_params()
	})
}

func Prove(lurkProgram string, privateParams Parameters, publicParams Parameters) ([]byte, error) {
	priv, err := privateParams.ToExpr()
	if err != nil {
		return nil, err
	}
	pub, err := publicParams.ToExpr()
	if err != nil {
		return nil, err
	}
	proof, tag, output, err := createProof(lurkProgram, priv, pub)
	if err != nil {
		return nil, err
	}
	if tag != TagSym || !bytes.Equal(output, OutputTrue) {
		return nil, errors.New("program output is not true")
	}

	return proof, nil
}

func Verify(lurkProgram string, publicParams Parameters, proof []byte) (bool, error) {
	pub, err := publicParams.ToExpr()
	if err != nil {
		return false, err
	}
	tagBytes := make([]byte, 32)
	tagBytes[len(tagBytes)-1] = byte(TagSym)
	return verifyProof(lurkProgram, pub, proof, tagBytes, OutputTrue)
}

func Eval(lurkProgram string, privateParams Parameters, publicParams Parameters) (Tag, []byte, int, error) {
	priv, err := privateParams.ToExpr()
	if err != nil {
		return TagNil, nil, 0, err
	}
	pub, err := publicParams.ToExpr()
	if err != nil {
		return TagNil, nil, 0, err
	}
	return evaluate(lurkProgram, priv, pub)
}

func createProof(lurkProgram, privateParams, publicParams string) ([]byte, Tag, []byte, error) {
	clurkProgram := C.CString(lurkProgram)
	cprivateParams := C.CString(privateParams)
	cpublicParams := C.CString(publicParams)

	defer C.free(unsafe.Pointer(clurkProgram))
	defer C.free(unsafe.Pointer(cprivateParams))
	defer C.free(unsafe.Pointer(cpublicParams))

	// Fixme: set to actual proof size
	var (
		proof     [12000]byte
		proofLen  C.size_t
		outputTag [32]byte
		outputVal [32]byte
	)

	result := C.create_proof_ffi(
		clurkProgram,
		cprivateParams,
		cpublicParams,
		(*C.uint8_t)(unsafe.Pointer(&proof[0])),
		&proofLen,
		(*C.uint8_t)(unsafe.Pointer(&outputTag[0])),
		(*C.uint8_t)(unsafe.Pointer(&outputVal[0])),
	)

	if result != 0 {
		return nil, TagNil, nil, errors.New("failed to create proof")
	}

	var (
		proofOut = make([]byte, proofLen)
		tagOut   = make([]byte, 32)
		valOut   = make([]byte, 32)
	)
	copy(proofOut, proof[:proofLen])
	copy(tagOut, outputTag[:32])
	copy(valOut, outputVal[:32])

	tag, err := TagFromBytes(tagOut)
	if err != nil {
		return nil, TagNil, nil, err
	}

	return proofOut, tag, valOut, nil
}

func verifyProof(lurkProgram, publicParams string, proof, expectedTag, expectedOutput []byte) (bool, error) {
	if len(proof) == 0 {
		return false, errors.New("proof is nil")
	}
	clurkProgram := C.CString(lurkProgram)
	cpublicParams := C.CString(publicParams)

	defer C.free(unsafe.Pointer(clurkProgram))
	defer C.free(unsafe.Pointer(cpublicParams))

	proofCopy := make([]byte, len(proof))
	copy(proofCopy[:], proof[:])

	tagCopy := make([]byte, 32)
	copy(tagCopy, expectedTag)

	outputCopy := make([]byte, 32)
	copy(outputCopy, expectedOutput)

	// Convert the Go byte slice to a C byte pointer
	cBytesProof := (*C.uint8_t)(unsafe.Pointer(&proofCopy[0]))
	proofSize := C.size_t(len(proofCopy))
	cBytesTag := (*C.uint8_t)(unsafe.Pointer(&tagCopy[0]))
	cBytesOutput := (*C.uint8_t)(unsafe.Pointer(&outputCopy[0]))

	result := C.verify_proof_ffi(
		clurkProgram,
		cpublicParams,
		cBytesProof,
		proofSize,
		cBytesTag,
		cBytesOutput,
	)

	if result < 0 {
		return false, errors.New("proof verification errored")
	}
	return result == 0, nil
}

func evaluate(lurkProgram, privateParams, publicParams string) (Tag, []byte, int, error) {
	clurkProgram := C.CString(lurkProgram)
	cprivateParams := C.CString(privateParams)
	cpublicParams := C.CString(publicParams)

	defer C.free(unsafe.Pointer(clurkProgram))
	defer C.free(unsafe.Pointer(cprivateParams))
	defer C.free(unsafe.Pointer(cpublicParams))

	// Fixme: set to actual proof size
	var (
		iterations C.size_t
		outputTag  [32]byte
		outputVal  [32]byte
	)

	result := C.eval_ffi(
		clurkProgram,
		cprivateParams,
		cpublicParams,
		(*C.uint8_t)(unsafe.Pointer(&outputTag[0])),
		(*C.uint8_t)(unsafe.Pointer(&outputVal[0])),
		&iterations,
	)

	if result != 0 {
		return TagNil, nil, 0, errors.New("failed to create proof")
	}

	var (
		tagOut = make([]byte, 32)
		valOut = make([]byte, 32)
	)
	copy(tagOut, outputTag[:32])
	copy(valOut, outputVal[:32])
	iter_out := iterations

	tag, err := TagFromBytes(tagOut)
	if err != nil {
		return TagNil, nil, 0, err
	}

	return tag, valOut, int(iter_out), nil
}
