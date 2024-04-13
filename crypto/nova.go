//go:build !skiprusttests
// +build !skiprusttests

// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

/*
#cgo linux CFLAGS: -Irust/target/release -Irust-target-release-crypto
#cgo linux LDFLAGS: -Lrust/target/release -Lrust-target-release-crypto -l:libillium_crypto.a -ldl -lpthread -lgcc_s -lc -lm -lssl -lcrypto -lstdc++
#cgo windows CFLAGS: -Irust/target/release
#cgo windows LDFLAGS: -Lrust/target/release -l:illium_crypto.lib
#cgo darwin CFLAGS: -Irust/target/release -Irust/target/aarch64-apple-darwin/release
#cgo darwin LDFLAGS: -Lrust/target/release -Irust/target/aarch64-apple-darwin/release -lillium_crypto -lc++ -lssl -lcrypto -framework SystemConfiguration
#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>
void generate_secret_key(uint8_t* out);
void secret_key_from_seed(const uint8_t* seed, uint8_t* out);
void priv_to_pub(const uint8_t* bytes, uint8_t* out);
void compressed_to_full(const uint8_t* bytes, uint8_t* out_x, uint8_t* out_y);
void sign(const uint8_t* privkey, const uint8_t* message_digest, uint8_t* out);
bool verify(const uint8_t* pubkey, const uint8_t* message_digest, const uint8_t* sig_r, const uint8_t* sig_s);
*/
import "C"
import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/crypto/pb"
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

// NovaPrivateKey is a Grumpkin curve private key in the nova proving system.
type NovaPrivateKey struct {
	k *[64]byte
}

// NovaPublicKey is a Grumpkin curve public key in the nova proving system.
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

// NewNovaKeyFromSeed deterministically derives a nova private key from a seed
func NewNovaKeyFromSeed(seed [32]byte) (crypto.PrivKey, crypto.PubKey, error) {
	priv := novaSecretKeyFromSeed(seed)
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

// Sign returns a signature from an input message. Note that this
// method expects a 32 byte digest of the raw data to sign and not
// the raw data itself. The passed in digest will not be hashed.
func (k *NovaPrivateKey) Sign(digest []byte) ([]byte, error) {
	var m [32]byte
	copy(m[:], digest)
	var mReversed [32]byte
	copy(mReversed[:], reverseBytes(m[:]))

	var sk [32]byte
	copy(sk[:], k.k[:NovaPrivateKeySize])
	sig := novaSign(sk, mReversed)
	return sig[:], nil
}

// PublicKeyFromXY builds a PublicKey from the x and y coordinates
func PublicKeyFromXY(x, y []byte) (crypto.PubKey, error) {
	if len(x) != 32 || len(y) != 32 {
		return nil, errors.New("invalid coordinate")
	}
	x2 := make([]byte, len(x))
	y2 := make([]byte, len(y))
	copy(x2, x)
	copy(y2, y)
	reverseBytes(x2)
	reverseBytes(y2)

	b := y2[0] & 0x01
	b <<= 6

	x2[31] |= b

	var compressed [32]byte
	copy(compressed[:], reverseBytes(x2))

	return &NovaPublicKey{
		k: &compressed,
	}, nil
}

// Type of the public key (Nova).
func (k *NovaPublicKey) Type() pb.KeyType {
	return Libp2pKeyTypeNova
}

// Raw public key bytes.
func (k *NovaPublicKey) Raw() ([]byte, error) {
	return k.k[:], nil
}

// ToXY returns the x and y coordinates of the PublicKey
func (k *NovaPublicKey) ToXY() ([]byte, []byte) {
	var pk [32]byte
	copy(pk[:], k.k[:])
	var pkReversed [32]byte
	copy(pkReversed[:], reverseBytes(pk[:]))
	x, y := compressed_to_full(pkReversed)
	return reverseBytes(x[:]), reverseBytes(y[:])
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
func (k *NovaPublicKey) Verify(digest []byte, sig []byte) (bool, error) {
	var m [32]byte
	copy(m[:], digest)

	var mReversed [32]byte
	copy(mReversed[:], reverseBytes(m[:]))

	var signature [64]byte
	copy(signature[:], sig)

	var pk [32]byte
	copy(pk[:], k.k[:])

	var pkReversed [32]byte
	copy(pkReversed[:], reverseBytes(pk[:]))

	valid := novaVerify(pk, mReversed, signature)
	return valid, nil
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
		pubkey = novaPrivToPub(privkey)
		copy(privKey[:NovaPrivateKeySize], privkey[:])
		copy(privKey[NovaPrivateKeySize:], pubkey[:])
	default:
		return nil, fmt.Errorf(
			"expected Nova data size to be %d or %d, got %d",
			NovaPrivateKeySize,
			NovaPrivateKeySize+NovaPublicKeySize,
			len(data),
		)
	}

	return &NovaPrivateKey{
		k: &privKey,
	}, nil
}

// UnmarshalSignature unmarshals a compressed signature into an uncompressed
// signature consisting of the x and y coordinates for the r value along with s.
func UnmarshalSignature(sig []byte) (sigRx, sigRy, sigS []byte) {
	r, s := [32]byte{}, [32]byte{}
	copy(r[:], sig[:32])
	copy(s[:], sig[32:])
	rx, ry := compressed_to_full(r)
	return reverseBytes(rx[:]), reverseBytes(ry[:]), s[:]
}

func novaGenerateSecretKey() [32]byte {
	var secretKey [32]byte

	// Call the Rust function to generate the secret key
	C.generate_secret_key((*C.uint8_t)(unsafe.Pointer(&secretKey[0])))

	return secretKey
}

func novaSecretKeyFromSeed(seed [32]byte) [32]byte {
	// Allocate memory for the secret key
	var secretKey [32]byte

	var seedCopy [32]byte
	copy(seedCopy[:], seed[:])

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&seedCopy[0]))

	// Call the Rust function to generate the secret key
	C.secret_key_from_seed(cBytes, (*C.uint8_t)(unsafe.Pointer(&secretKey[0])))

	return secretKey
}

func novaPrivToPub(sk [32]byte) [32]byte {
	var pubkey [32]byte

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&sk[0]))

	// Call the Rust function to compute the public key
	C.priv_to_pub(cBytes, (*C.uint8_t)(unsafe.Pointer(&pubkey[0])))

	var reversed [32]byte
	copy(reversed[:], reverseBytes(pubkey[:]))

	return reversed
}

func compressed_to_full(pk [32]byte) ([32]byte, [32]byte) {
	var x [32]byte
	var y [32]byte

	// Convert the Go byte slice to a C byte pointer
	cBytes := (*C.uint8_t)(unsafe.Pointer(&pk[0]))

	// Call the Rust function to compute the public key
	C.compressed_to_full(cBytes, (*C.uint8_t)(unsafe.Pointer(&x[0])), (*C.uint8_t)(unsafe.Pointer(&y[0])))

	return x, y
}

func novaSign(sk [32]byte, messageDigest [32]byte) [64]byte {
	var signature [64]byte

	// Convert the Go byte slices to C byte pointers
	cPrivBytes := (*C.uint8_t)(unsafe.Pointer(&sk[0]))
	cDigestBytes := (*C.uint8_t)(unsafe.Pointer(&messageDigest[0]))

	// Call the Rust function
	C.sign(cPrivBytes, cDigestBytes, (*C.uint8_t)(unsafe.Pointer(&signature[0])))

	return signature
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

func reverseBytes(b []byte) []byte {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b
}
