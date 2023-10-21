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

void generate_secret_key(uint8_t* out);

void priv_to_pub(const uint8_t* bytes, uint8_t* out);

void sign(const uint8_t* privkey, const uint8_t* message_digest, uint8_t* out);

bool verify(const uint8_t* pubkey, const uint8_t* message_digest, const uint8_t* sig_r, const uint8_t* sig_s);

// Helper function to free memory allocated by Rust
void free_memory(void* ptr);
*/
import "C"
import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/crypto/pb"
	"github.com/project-illium/ilxd/params/hash"
	"golang.org/x/crypto/curve25519"
	"io"
	"unsafe"
)

const (
	Libp2pKeyTypeNova  = pb.KeyType(5)
	NovaPrivateKeySize = 32
	NovaPublicKeySize  = 32
)

func init() {
	crypto.PubKeyUnmarshallers[Libp2pKeyTypeNova] = UnmarshalNovaPublicKey
	crypto.PrivKeyUnmarshallers[Libp2pKeyTypeNova] = UnmarshalNovaPrivateKey
}

// NovaPrivateKey is a Vesta curve private key in the nova proving system.
type NovaPrivateKey struct {
	k *[64]byte
}

// NovaPublicKey is a Vesta curve public key in the nova proving system.
type NovaPublicKey struct {
	k *[32]byte
}

// GenerateNovaKey generates a new Nova private and public key pair.
func GenerateNovaKey(src io.Reader) (crypto.PrivKey, crypto.PubKey, error) {
	priv := novaGenerateSecretKey()
	pub := novaPrivToPub(priv)

	var combined [64]byte
	copy(combined[:32], priv[:])
	copy(combined[32:], pub[:])

	return &NovaPrivateKey{
			k: &combined,
		},
		&NovaPublicKey{
			k: &pub,
		},
		nil
}

// Type of the private key (Curve25519).
func (k *NovaPrivateKey) Type() pb.KeyType {
	return Libp2pKeyTypeNova
}

// Raw private key bytes.
func (k *NovaPrivateKey) Raw() ([]byte, error) {
	// The Nova private key contains two 32-bytes curve points, the private
	// key and the public key.
	// It makes it more efficient to get the public key without re-computing an
	// elliptic curve multiplication.
	buf := make([]byte, len(k.k))
	copy(buf, k.k[:])

	return buf, nil
}

func (k *NovaPrivateKey) pubKeyBytes() []byte {
	return k.k[NovaPrivateKeySize:]
}

// nolint:unused
func (k *NovaPrivateKey) privKeyBytes() []byte {
	return k.k[:NovaPrivateKeySize]
}

// Equals compares two Nova private keys.
func (k *NovaPrivateKey) Equals(o crypto.Key) bool {
	cdk, ok := o.(*NovaPrivateKey)
	if !ok {
		return basicEquals(k, o)
	}

	return subtle.ConstantTimeCompare(k.k[:], cdk.k[:]) == 1
}

// GetPublic returns an Nova public key from a private key.
func (k *NovaPrivateKey) GetPublic() crypto.PubKey {
	var pubkey [32]byte
	copy(pubkey[:], k.pubKeyBytes())

	return &NovaPublicKey{k: &pubkey}
}

// Sign returns a signature from an input message.
func (k *NovaPrivateKey) Sign(msg []byte) ([]byte, error) {
	h := hash.HashFunc(msg)

	var m [32]byte
	copy(m[:], h)

	var sk [32]byte
	copy(sk[:], k.k[:NovaPrivateKeySize])
	sig := novaSign(sk, m)
	return sig[:], nil
}

// Type of the public key (Nova).
func (k *NovaPublicKey) Type() pb.KeyType {
	return Libp2pKeyTypeNova
}

// Raw public key bytes.
func (k *NovaPublicKey) Raw() ([]byte, error) {
	return k.k[:], nil
}

// Equals compares two Nova public keys.
func (k *NovaPublicKey) Equals(o crypto.Key) bool {
	edk, ok := o.(*NovaPublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return bytes.Equal(k.k[:], edk.k[:])
}

// Verify checks a signature agains the input data.
func (k *NovaPublicKey) Verify(data []byte, sig []byte) (bool, error) {
	h := hash.HashFunc(data)

	var m [32]byte
	copy(m[:], h)

	var signature [64]byte
	copy(signature[:], sig)

	valid := novaVerify(*k.k, m, signature)
	return valid, nil
}

// Encrypt encrypts the plaintext using the public key and returns the ciphertext.
func (k *NovaPublicKey) Encrypt(plaintext []byte) ([]byte, error) {
	return Encrypt(k, plaintext)
}

// UnmarshalNovaPublicKey returns a public key from input bytes.
func UnmarshalNovaPublicKey(data []byte) (crypto.PubKey, error) {
	if len(data) != 32 {
		return nil, errors.New("expect Nova public key data size to be 32")
	}

	var pubkey [32]byte
	copy(pubkey[:], data)

	return &NovaPublicKey{
		k: &pubkey,
	}, nil
}

// UnmarshalNovaPrivateKey returns a private key from input bytes.
func UnmarshalNovaPrivateKey(data []byte) (crypto.PrivKey, error) {
	var privKey [64]byte
	switch len(data) {
	case NovaPrivateKeySize + NovaPublicKeySize:
		// Remove the redundant public key. See issue #36.
		redundantPk := data[NovaPrivateKeySize:]
		pk := data[NovaPrivateKeySize : NovaPrivateKeySize+NovaPublicKeySize]
		if subtle.ConstantTimeCompare(pk, redundantPk) == 0 {
			return nil, errors.New("expected redundant Nova public key to be redundant")
		}
		copy(privKey[:], data)
	case NovaPrivateKeySize:
		var (
			pubkey  [32]byte
			privkey [32]byte
		)
		copy(privkey[:], data[:NovaPrivateKeySize])
		curve25519.ScalarBaseMult(&pubkey, &privkey)
		copy(privKey[:NovaPrivateKeySize], privkey[:])
		copy(privKey[NovaPrivateKeySize:], pubkey[:])
	default:
		return nil, fmt.Errorf(
			"expected Curve25519 data size to be %d or %d, got %d",
			NovaPrivateKeySize,
			NovaPrivateKeySize+NovaPublicKeySize,
			len(data),
		)
	}

	return &NovaPrivateKey{
		k: &privKey,
	}, nil
}

func novaGenerateSecretKey() [32]byte {
	// Define the length of the secret key (32 bytes)
	secretKeyLen := 32

	// Allocate memory for the secret key
	secretKey := make([]byte, secretKeyLen)

	// Call the Rust function to generate the secret key
	C.generate_secret_key((*C.uint8_t)(unsafe.Pointer(&secretKey[0])))

	var ret [32]byte
	copy(ret[:], secretKey)
	return ret
}

func novaPrivToPub(pk [32]byte) [32]byte {
	// Create a byte slice for the result
	pubkey := make([]byte, 32)

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&pk[0]))

	// Call the Rust function to compute the public key
	C.priv_to_pub(cBytes, (*C.uint8_t)(unsafe.Pointer(&pubkey[0])))

	var ret [32]byte
	copy(ret[:], pubkey)
	return ret
}

func novaSign(sk [32]byte, messageDigest [32]byte) [64]byte {
	// Ensure that the provided signature buffer is large enough
	signature := make([]byte, 64)

	// Convert the Go byte slices to C byte pointers
	cPrivBytes := (*C.uint8_t)(unsafe.Pointer(&sk[0]))
	cDigestBytes := (*C.uint8_t)(unsafe.Pointer(&messageDigest[0]))

	// Call the Rust function
	C.sign(cPrivBytes, cDigestBytes, (*C.uint8_t)(unsafe.Pointer(&signature[0])))

	var ret [64]byte
	copy(ret[:], signature)
	return ret
}

func novaVerify(pk [32]byte, messageDigest [32]byte, signature [64]byte) bool {
	// Convert the Go byte slices to C byte pointers
	cPubBytes := (*C.uint8_t)(unsafe.Pointer(&pk[0]))
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
