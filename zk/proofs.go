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
    size_t proof_size);
*/
import "C"
import (
	"bytes"
	"errors"
	"sync"
	"unsafe"
)

var once sync.Once

var trueVal = []byte{4, 200, 219, 197, 70, 43, 38, 162, 69, 194, 112, 0, 19, 99, 1, 240, 70, 225, 92, 182, 50, 158, 201, 58, 139, 58, 93, 133, 73, 161, 152, 86}

// LoadZKPublicParameters loads the lurk public parameters from disk
// into memory or generates them if this is the first startup.
func LoadZKPublicParameters() {
	once.Do(func() {
		log.Info("Loading lurk public parameters...")
		C.load_public_params()
	})
}

func Prove(lurkProgram string, privateParams Parameters, publicParams Parameters) ([]byte, error) {
	proof, tag, output, err := createProof(lurkProgram, privateParams.ToExpr(), publicParams.ToExpr())
	if err != nil {
		return nil, err
	}
	if tag != TagSym || !bytes.Equal(output, trueVal) {
		return nil, errors.New("program output is not true")
	}

	return proof, nil
}

func Verify(lurkProgram string, publicParams Parameters, proof []byte) (bool, error) {
	return verifyProof(lurkProgram, publicParams.ToExpr(), proof)
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

func verifyProof(lurkProgram, publicParams string, proof []byte) (bool, error) {
	clurkProgram := C.CString(lurkProgram)
	cpublicParams := C.CString(publicParams)

	defer C.free(unsafe.Pointer(clurkProgram))
	defer C.free(unsafe.Pointer(cpublicParams))

	proofCopy := make([]byte, len(proof))
	copy(proofCopy[:], proof[:])

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&proofCopy[0]))
	proofSize := C.size_t(len(proofCopy))

	result := C.verify_proof_ffi(
		clurkProgram,
		cpublicParams,
		cBytes,
		proofSize,
	)

	if result < 0 {
		return false, errors.New("proof verification errored")
	}
	return result == 0, nil
}
