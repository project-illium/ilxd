// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/sha512"
	"filippo.io/edwards25519"
)

// privateKeyToCurve25519 converts an ed25519 private key into a corresponding
// curve25519 private key such that the resulting curve25519 public key will
// equal the result from PublicKeyToCurve25519.
func privateKeyToCurve25519(curve25519Private *[32]byte, privateKey *[64]byte) {
	h := sha512.New()
	h.Write(privateKey[:32])
	digest := h.Sum(nil)

	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	copy(curve25519Private[:], digest)
}

// publicKeyToCurve25519 converts an Ed25519 public key into the curve25519
// public key that would be generated from the same private key.
func publicKeyToCurve25519(curve25519Public *[32]byte, publicKey *[32]byte) error {
	point, err := new(edwards25519.Point).SetBytes(publicKey[:])
	if err != nil {
		return err
	}
	copy(curve25519Public[:], point.BytesMontgomery())
	return nil
}
