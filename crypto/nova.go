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

void generate_secret_key(uint8_t* out);
size_t key_len();

void priv_to_pub(const uint8_t* bytes, uint8_t* out);

void sign(const uint8_t* privkey, const uint8_t* message_digest, uint8_t* out);

bool verify(const uint8_t* pubkey, const uint8_t* message_digest, const uint8_t* sig_r, const uint8_t* sig_s);

// Helper function to free memory allocated by Rust
void free_memory(void* ptr);
*/
import "C"
import (
	"unsafe"
)

func NewSecretKey() []byte {
	// Define the length of the secret key (32 bytes)
	secretKeyLen := 32

	// Allocate memory for the secret key
	secretKey := make([]byte, secretKeyLen)

	// Call the Rust function to generate the secret key
	C.generate_secret_key((*C.uint8_t)(unsafe.Pointer(&secretKey[0])))

	return secretKey
}

func PrivToPub(pk []byte) []byte {
	// Create a byte slice for the result
	result := make([]byte, 32)

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&pk[0]))

	// Call the Rust function to compute the public key
	C.priv_to_pub(cBytes, (*C.uint8_t)(unsafe.Pointer(&result[0])))

	return result
}

func Sign(privKey []byte, messageDigest []byte) []byte {
	// Ensure that the provided signature buffer is large enough
	signature := make([]byte, 64)

	// Convert the Go byte slices to C byte pointers
	cPrivBytes := (*C.uint8_t)(unsafe.Pointer(&privKey[0]))
	cDigestBytes := (*C.uint8_t)(unsafe.Pointer(&messageDigest[0]))

	// Call the Rust function
	C.sign(cPrivBytes, cDigestBytes, (*C.uint8_t)(unsafe.Pointer(&signature[0])))

	return signature
}

func Verify(pubKey []byte, messageDigest []byte, signature []byte) bool {
	// Convert the Go byte slices to C byte pointers
	cPubBytes := (*C.uint8_t)(unsafe.Pointer(&pubKey[0]))
	cDigestBytes := (*C.uint8_t)(unsafe.Pointer(&messageDigest[0]))

	// Split the signature into sigR and sigS
	sigR := signature[:32]
	sigS := signature[32:]

	// Convert the Go byte slices for sigR and sigS to C byte pointers
	cSigRBytes := (*C.uint8_t)(unsafe.Pointer(&sigR[0]))
	cSigSBytes := (*C.uint8_t)(unsafe.Pointer(&sigS[0]))

	// Call the Rust function
	valid := C.verify(cPubBytes, cDigestBytes, cSigRBytes, cSigSBytes)

	return bool(valid)
}
