// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

/*
#cgo CFLAGS: -Irust/target/debug
#cgo LDFLAGS: -Lrust/target/debug -lillium_crypto -ldl -lpthread -lgcc_s -lc -lm

#include <stdint.h>
#include <stddef.h>

// Define the Rust function's C signature
struct SecretKeyBytes {
    const uint8_t* data;
    size_t len;
};

struct SecretKeyBytes generate_secret_key();
size_t secret_key_len();

// Helper function to free memory allocated by Rust
void free_memory(void* ptr);
*/
import "C"
import (
	"unsafe"
)

func NewSecretKey() []byte {
	// Call the Rust function to generate the secret key
	cSecretKey := C.generate_secret_key()

	// Convert the C struct to a pointer
	secretKey := (*C.struct_SecretKeyBytes)(unsafe.Pointer(&cSecretKey))

	// Convert the C array to a Go byte slice
	keySlice := C.GoBytes(unsafe.Pointer(secretKey.data), C.int(secretKey.len))

	// Free the memory allocated by Rust
	C.free_memory(unsafe.Pointer(secretKey.data))

	return []byte(keySlice)
}
