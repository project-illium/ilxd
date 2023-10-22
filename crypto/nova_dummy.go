//go:build skiprusttests
// +build skiprusttests

// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/ed25519"
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
type NovaPrivateKey crypto.Ed25519PrivateKey

// NovaPublicKey is a Vesta curve public key in the nova proving system.
type NovaPublicKey crypto.Ed25519PublicKey

// GenerateNovaKey generates a new Nova private and public key pair.
func GenerateNovaKey(src io.Reader) (crypto.PrivKey, crypto.PubKey, error) {
	return crypto.GenerateEd25519Key(src)
}

// NewNovaKeyFromSeed deterministically derives a nova private key from a seed
func NewNovaKeyFromSeed(seed [32]byte) (crypto.PrivKey, crypto.PubKey, error) {
	priv := ed25519.NewKeyFromSeed(seed[:])
	privKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	return privKey, privKey.GetPublic(), nil
}

// Type of the private key (Curve25519).
func (k *NovaPrivateKey) Type() pb.KeyType {
	return Libp2pKeyTypeNova
}

// Type of the public key (Nova).
func (k *NovaPublicKey) Type() pb.KeyType {
	return Libp2pKeyTypeNova
}

// UnmarshalNovaPublicKey returns a public key from input bytes.
func UnmarshalNovaPublicKey(data []byte) (crypto.PubKey, error) {
	return crypto.UnmarshalEd25519PublicKey(data)
}

// UnmarshalNovaPrivateKey returns a private key from input bytes.
func UnmarshalNovaPrivateKey(data []byte) (crypto.PrivKey, error) {
	return crypto.UnmarshalEd25519PrivateKey(data)
}
