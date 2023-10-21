// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

/*
#cgo CFLAGS: -Irust/target/release
#cgo LDFLAGS: -Lrust/target/release -lillium_crypto -ldl -lpthread -lgcc_s -lc -lm

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

// Define the Rust function's C signature
struct SerializedBytes {
    const uint8_t* data;
    size_t len;
};

struct SerializedBytes generate_secret_key();
size_t key_len();

struct SerializedBytes priv_to_pub(const uint8_t* bytes, size_t len);

struct SerializedBytes sign(const uint8_t* priv_bytes, size_t priv_len, const uint8_t* digest_bytes, size_t digest_len);

bool verify(const uint8_t* pub_bytes, size_t pub_len, const uint8_t* digest_bytes, size_t digest_len, const uint8_t* sigr_bytes, size_t sigr_len, const uint8_t* sigs_bytes, size_t sigs_len);

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
	secretKey := (*C.struct_SerializedBytes)(unsafe.Pointer(&cSecretKey))

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

func Sign(privKey []byte, messageDigest []byte) []byte {
	priv_len := len(privKey)

	// Convert the Go byte slice to a C byte pointer
	cPrivBytes := (*C.uint8_t)(unsafe.Pointer(&privKey[0]))

	digest_len := len(messageDigest)

	// Convert the Go byte slice to a C byte pointer
	cDigestBytes := (*C.uint8_t)(unsafe.Pointer(&messageDigest[0]))

	// Call the Rust function
	signature := C.sign(cPrivBytes, C.size_t(priv_len), cDigestBytes, C.size_t(digest_len))

	// Convert the returned data to a Go byte slice
	sigData := C.GoBytes(unsafe.Pointer(signature.data), C.int(signature.len))

	// Free the memory allocated by Rust
	C.free_memory(unsafe.Pointer(signature.data))

	return []byte(sigData)
}

func Verify(pubKey []byte, messageDigest []byte, signature []byte) bool {
	pub_len := len(pubKey)

	// Convert the Go byte slice to a C byte pointer
	cPubBytes := (*C.uint8_t)(unsafe.Pointer(&pubKey[0]))

	digest_len := len(messageDigest)

	// Convert the Go byte slice to a C byte pointer
	cDigestBytes := (*C.uint8_t)(unsafe.Pointer(&messageDigest[0]))

	sigR := signature[:32]
	sigS := signature[32:]

	sigr_len := 32

	// Convert the Go byte slice to a C byte pointer
	cSigRBytes := (*C.uint8_t)(unsafe.Pointer(&sigR[0]))

	sigs_len := 32

	// Convert the Go byte slice to a C byte pointer
	cSigSBytes := (*C.uint8_t)(unsafe.Pointer(&sigS[0]))

	// Call the Rust function
	valid := C.verify(cPubBytes, C.size_t(pub_len), cDigestBytes, C.size_t(digest_len), cSigRBytes, C.size_t(sigr_len), cSigSBytes, C.size_t(sigs_len))

	// Free the memory allocated by Rust
	//C.free_memory(unsafe.Pointer(valid))

	return bool(valid)
}
