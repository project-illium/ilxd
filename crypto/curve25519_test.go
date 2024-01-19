// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/rand"
	"github.com/go-test/deep"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCurve25519(t *testing.T) {
	priv, pub, err := GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	privBytes, err := crypto.MarshalPrivateKey(priv)
	assert.NoError(t, err)

	pubBytes, err := crypto.MarshalPublicKey(pub)
	assert.NoError(t, err)

	priv2, err := crypto.UnmarshalPrivateKey(privBytes)
	assert.NoError(t, err)

	pub2, err := crypto.UnmarshalPublicKey(pubBytes)
	assert.NoError(t, err)

	assert.Empty(t, deep.Equal(priv, priv2))
	assert.Empty(t, deep.Equal(pub, pub2))

	edPriv, edPub, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	curvePriv, err := Curve25519PrivateKeyFromEd25519(edPriv)
	assert.NoError(t, err)

	curvePub, err := Curve25519PublicKeyFromEd25519(edPub)
	assert.NoError(t, err)

	message := []byte("message")
	cipherText, err := Encrypt(curvePub, message)
	assert.NoError(t, err)

	plainText, err := Decrypt(curvePriv, cipherText)
	assert.NoError(t, err)

	assert.Equal(t, message, plainText)

	curvePub2 := curvePriv.GetPublic()

	cipherText, err = Encrypt(curvePub2, message)
	assert.NoError(t, err)

	plainText, err = Decrypt(curvePriv, cipherText)
	assert.NoError(t, err)

	assert.Equal(t, message, plainText)

	assert.False(t, curvePriv.Equals(priv))
	assert.False(t, curvePub.Equals(pub))

	var seed [32]byte
	rand.Read(seed[:])

	priv4, _, err := NewCurve25519KeyFromSeed(seed)
	assert.NoError(t, err)

	priv5, _, err := NewCurve25519KeyFromSeed(seed)
	assert.NoError(t, err)

	assert.True(t, priv4.Equals(priv5))
}
