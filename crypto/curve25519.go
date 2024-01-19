// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	pb "github.com/libp2p/go-libp2p/core/crypto/pb"
	"github.com/nixberg/chacha-rng-go"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
	"io"
)

const (
	Libp2pKeyTypeCurve25519  = pb.KeyType(4)
	Curve25519PrivateKeySize = 32
	Curve25519PublicKeySize  = 32
)

func init() {
	crypto.PubKeyUnmarshallers[Libp2pKeyTypeCurve25519] = UnmarshalCurve25519PublicKey
	crypto.PrivKeyUnmarshallers[Libp2pKeyTypeCurve25519] = UnmarshalCurve25519PrivateKey
}

var ErrSigNoop = errors.New("curve25519 keys cannot do signing or verification")

// Curve25519PrivateKey is a Curve25519 private key.
type Curve25519PrivateKey struct {
	k *[64]byte
}

// Curve25519PublicKey is a Curve25519 public key.
type Curve25519PublicKey struct {
	k *[32]byte
}

// GenerateCurve25519Key generates a new Curve25519 private and public key pair.
func GenerateCurve25519Key(src io.Reader) (crypto.PrivKey, crypto.PubKey, error) {
	pub, priv, err := box.GenerateKey(src)
	if err != nil {
		return nil, nil, err
	}

	var combined [64]byte
	copy(combined[:32], priv[:])
	copy(combined[32:], pub[:])

	return &Curve25519PrivateKey{
			k: &combined,
		},
		&Curve25519PublicKey{
			k: pub,
		},
		nil
}

type chachaRrng struct {
	rng *chacha.ChaCha
}

func (c *chachaRrng) Read(p []byte) (n int, err error) {
	remaining := len(p)
	cursor := 0

	for remaining > 0 {
		val := c.rng.Uint64()
		if remaining < 8 {
			binary.LittleEndian.PutUint64(p[cursor:], val)
			cursor += remaining
			break
		} else {
			binary.LittleEndian.PutUint64(p[cursor:], val)
			cursor += 8
			remaining -= 8
		}
	}

	if cursor == 0 {
		return 0, errors.New("unable to fill byte slice")
	}

	return cursor, nil
}

func NewCurve25519KeyFromSeed(seed [32]byte) (crypto.PrivKey, crypto.PubKey, error) {
	var s [8]uint32
	for i := 0; i < 8; i++ {
		s[i] = binary.LittleEndian.Uint32(seed[i*4 : (i+1)*4])
	}
	rng := chachaRrng{chacha.Seeded20(s, 0)}

	pub, priv, err := box.GenerateKey(&rng)
	if err != nil {
		return nil, nil, err
	}

	var combined [64]byte
	copy(combined[:32], priv[:])
	copy(combined[32:], pub[:])

	return &Curve25519PrivateKey{
			k: &combined,
		},
		&Curve25519PublicKey{
			k: pub,
		},
		nil
}

func Curve25519PrivateKeyFromEd25519(privKey crypto.PrivKey) (crypto.PrivKey, error) {
	raw, err := privKey.Raw()
	if err != nil {
		return nil, err
	}
	var (
		curve25519Key [32]byte
		ed25519Key    [64]byte
	)
	copy(ed25519Key[:], raw)
	privateKeyToCurve25519(&curve25519Key, &ed25519Key)
	return UnmarshalCurve25519PrivateKey(curve25519Key[:])
}

func Curve25519PublicKeyFromEd25519(pubKey crypto.PubKey) (crypto.PubKey, error) {
	raw, err := pubKey.Raw()
	if err != nil {
		return nil, err
	}
	var (
		curve25519Key [32]byte
		ed25519Key    [32]byte
	)
	copy(ed25519Key[:], raw)
	if err := publicKeyToCurve25519(&curve25519Key, &ed25519Key); err != nil {
		return nil, err
	}
	return UnmarshalCurve25519PublicKey(curve25519Key[:])
}

// Type of the private key (Curve25519).
func (k *Curve25519PrivateKey) Type() pb.KeyType {
	return Libp2pKeyTypeCurve25519
}

// Raw private key bytes.
func (k *Curve25519PrivateKey) Raw() ([]byte, error) {
	// The Curve25519 private key contains two 32-bytes curve points, the private
	// key and the public key.
	// It makes it more efficient to get the public key without re-computing an
	// elliptic curve multiplication.
	buf := make([]byte, len(k.k))
	copy(buf, k.k[:])

	return buf, nil
}

func (k *Curve25519PrivateKey) pubKeyBytes() []byte {
	return k.k[Curve25519PrivateKeySize:]
}

// nolint:unused
func (k *Curve25519PrivateKey) privKeyBytes() []byte {
	return k.k[:Curve25519PrivateKeySize]
}

// Equals compares two Curve25519 private keys.
func (k *Curve25519PrivateKey) Equals(o crypto.Key) bool {
	cdk, ok := o.(*Curve25519PrivateKey)
	if !ok {
		return basicEquals(k, o)
	}

	return subtle.ConstantTimeCompare(k.k[:], cdk.k[:]) == 1
}

// GetPublic returns an Curve25519 public key from a private key.
func (k *Curve25519PrivateKey) GetPublic() crypto.PubKey {
	var pubkey [32]byte
	copy(pubkey[:], k.pubKeyBytes())

	return &Curve25519PublicKey{k: &pubkey}
}

// Sign returns a signature from an input message.
// This is a noop.
func (k *Curve25519PrivateKey) Sign(msg []byte) ([]byte, error) {
	return nil, ErrSigNoop
}

// Decrypt attempts to decrypt ciphertext using the private key.
func (k *Curve25519PrivateKey) Decrypt(cipherText []byte) ([]byte, error) {
	return Decrypt(k, cipherText)
}

// Type of the public key (Curve25519).
func (k *Curve25519PublicKey) Type() pb.KeyType {
	return Libp2pKeyTypeCurve25519
}

// Raw public key bytes.
func (k *Curve25519PublicKey) Raw() ([]byte, error) {
	return k.k[:], nil
}

// Equals compares two Curve25519 public keys.
func (k *Curve25519PublicKey) Equals(o crypto.Key) bool {
	edk, ok := o.(*Curve25519PublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return bytes.Equal(k.k[:], edk.k[:])
}

// Verify checks a signature agains the input data.
// This is a noop.
func (k *Curve25519PublicKey) Verify(data []byte, sig []byte) (bool, error) {
	return false, ErrSigNoop
}

// Encrypt encrypts the plaintext using the public key and returns the ciphertext.
func (k *Curve25519PublicKey) Encrypt(plaintext []byte) ([]byte, error) {
	return Encrypt(k, plaintext)
}

// UnmarshalCurve25519PublicKey returns a public key from input bytes.
func UnmarshalCurve25519PublicKey(data []byte) (crypto.PubKey, error) {
	if len(data) != 32 {
		return nil, errors.New("expect Curve25519 public key data size to be 32")
	}

	var pubkey [32]byte
	copy(pubkey[:], data)

	return &Curve25519PublicKey{
		k: &pubkey,
	}, nil
}

// UnmarshalCurve25519PrivateKey returns a private key from input bytes.
func UnmarshalCurve25519PrivateKey(data []byte) (crypto.PrivKey, error) {
	var privKey [64]byte
	switch len(data) {
	case Curve25519PrivateKeySize + Curve25519PublicKeySize:
		// Remove the redundant public key. See issue #36.
		redundantPk := data[Curve25519PrivateKeySize:]
		pk := data[Curve25519PrivateKeySize : Curve25519PrivateKeySize+Curve25519PublicKeySize]
		if subtle.ConstantTimeCompare(pk, redundantPk) == 0 {
			return nil, errors.New("expected redundant Curve25519 public key to be redundant")
		}
		copy(privKey[:], data)
	case Curve25519PrivateKeySize:
		var (
			pubkey  [32]byte
			privkey [32]byte
		)
		copy(privkey[:], data[:Curve25519PrivateKeySize])
		curve25519.ScalarBaseMult(&pubkey, &privkey)
		copy(privKey[:Curve25519PrivateKeySize], privkey[:])
		copy(privKey[Curve25519PrivateKeySize:], pubkey[:])
	default:
		return nil, fmt.Errorf(
			"expected Curve25519 data size to be %d or %d, got %d",
			Curve25519PrivateKeySize,
			Curve25519PrivateKeySize+Curve25519PublicKeySize,
			len(data),
		)
	}

	return &Curve25519PrivateKey{
		k: &privKey,
	}, nil
}

func basicEquals(k1, k2 crypto.Key) bool {
	if k1.Type() != k2.Type() {
		return false
	}

	a, err := k1.Raw()
	if err != nil {
		return false
	}
	b, err := k2.Raw()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
