// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/rand"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/nacl/box"
)

const (
	// Length of nacl nonce
	NonceBytes = 24

	// Length of nacl ephemeral public key
	EphemeralPublicKeyBytes = 32
)

// ErrBoxDecryption Nacl box decryption failed
var ErrBoxDecryption = errors.New("failed to decrypt curve25519")

// Encrypt encrypts an output with the public key.
func Encrypt(pubKey crypto.PubKey, plaintext []byte) ([]byte, error) {
	curve25519PubKey, ok := pubKey.(*Curve25519PublicKey)
	if !ok {
		return nil, errors.New("pubkey must be of type Curve25519PublicKey")
	}

	// Encrypt with nacl
	var (
		ciphertext []byte
		pt         = make([]byte, len(plaintext))
		err        error
	)
	copy(pt, plaintext)

	ciphertext, err = box.SealAnonymous(ciphertext, pt, curve25519PubKey.k, rand.Reader)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// Decrypt decrypts an output using a private key.
func Decrypt(privKey crypto.PrivKey, ciphertext []byte) ([]byte, error) {
	curve25519PrivKey, ok := privKey.(*Curve25519PrivateKey)
	if !ok {
		return nil, errors.New("privkey must be of type Curve25519PrivateKey")
	}

	var (
		plaintext []byte
		priv      [32]byte
		pub       [32]byte
	)
	copy(priv[:], curve25519PrivKey.k[:Curve25519PrivateKeySize])
	copy(pub[:], curve25519PrivKey.pubKeyBytes())

	plaintext, success := box.OpenAnonymous(plaintext, ciphertext, &pub, &priv)
	if !success {
		return nil, ErrBoxDecryption
	}
	return plaintext, nil
}
