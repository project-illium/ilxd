// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

/*
#cgo CFLAGS: -Irust/target/release
#cgo LDFLAGS: -Lrust/target/release -lillium_crypto -ldl -lpthread -lgcc_s -lc -lm

#include <stdint.h>
#include <stddef.h>

// Define the Rust function's C signature
struct KeyBytes {
    const uint8_t* data;
    size_t len;
};

struct KeyBytes generate_secret_key();
size_t key_len();

struct KeyBytes priv_to_pub(const uint8_t* bytes, size_t len);

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
	secretKey := (*C.struct_KeyBytes)(unsafe.Pointer(&cSecretKey))

	// Convert the C array to a Go byte slice
	keyData := C.GoBytes(unsafe.Pointer(secretKey.data), C.int(secretKey.len))

	// Free the memory allocated by Rust
	C.free_memory(unsafe.Pointer(secretKey.data))

	return []byte(keyData)
}

func PrivToPub(data []byte) []byte {
	length := len(data)

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&data[0]))

	// Call the Rust function
	publicKey := C.priv_to_pub(cBytes, C.size_t(length))

	// Convert the returned data to a Go byte slice
	keyData := C.GoBytes(unsafe.Pointer(publicKey.data), C.int(publicKey.len))

	// Free the memory allocated by Rust
	C.free_memory(unsafe.Pointer(publicKey.data))

	return []byte(keyData)
}
