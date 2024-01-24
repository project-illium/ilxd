// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

/*
#include <stdlib.h>
extern int lurk_commit(const char* expr, unsigned char* out);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// CommitmentLen is the length of the Lurk Commitment
const CommitmentLen = 32

// LurkCommit returns poseidon hash of provided lurk expression. This
// is the same exact hashing algorithm used inside lurk circuits.
func LurkCommit(expr string) ([]byte, error) {
	var out [32]byte
	cExpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cExpr))

	// Call the Rust function
	ret := C.lurk_commit(cExpr, (*C.uchar)(unsafe.Pointer(&out[0])))
	if ret != 0 {
		return out[:], fmt.Errorf("lurk_commit failed")
	}

	var b [32]byte
	copy(b[:], out[:])
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b[:], nil
}
