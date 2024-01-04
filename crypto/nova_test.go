//go:build !skiprusttests
// +build !skiprusttests

// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/go-test/deep"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNova(t *testing.T) {
	priv, pub, err := GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	privBytes, err := crypto.MarshalPrivateKey(priv)
	assert.NoError(t, err)

	pubBytes, err := crypto.MarshalPublicKey(pub)
	assert.NoError(t, err)

	priv2, err := crypto.UnmarshalPrivateKey(privBytes)
	assert.NoError(t, err)

	pub2, err := crypto.UnmarshalPublicKey(pubBytes)
	assert.NoError(t, err)

	assert.True(t, priv.Equals(priv2))

	assert.Empty(t, deep.Equal(priv, priv2))
	assert.Empty(t, deep.Equal(pub, pub2))

	message := hash.HashFunc([]byte("message"))
	sig, err := priv.Sign(message)
	assert.NoError(t, err)

	valid, err := pub.Verify(message, sig)
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = pub.Verify([]byte("fake message"), sig)
	assert.NoError(t, err)
	assert.False(t, valid)

	pub3 := priv.GetPublic()

	valid, err = pub3.Verify(message, sig)
	assert.NoError(t, err)
	assert.True(t, valid)

	assert.True(t, priv.Equals(priv2))
	assert.True(t, pub.Equals(pub2))

	var seed [32]byte
	rand.Read(seed[:])

	priv4, _, err := NewNovaKeyFromSeed(seed)
	assert.NoError(t, err)

	priv5, _, err := NewNovaKeyFromSeed(seed)
	assert.NoError(t, err)

	assert.True(t, priv4.Equals(priv5))

	x, y := pub3.(*NovaPublicKey).ToXY()
	assert.NotNil(t, x)
	assert.NotNil(t, y)
}

func TestPublicKeyFromXY(t *testing.T) {
	_, key, err := GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	x, y := key.(*NovaPublicKey).ToXY()

	key2, err := PublicKeyFromXY(x, y)
	assert.NoError(t, err)

	assert.Equal(t, key.(*NovaPublicKey).k, key2.(*NovaPublicKey).k)
}

func TestUnmarshalNovaPrivateKey(t *testing.T) {
	s, err := hex.DecodeString("2ce70d73dc1a84a3d84a7301652c680b226b8e8da51fa9e518d436578c5e3427")
	assert.NoError(t, err)

	fmt.Println(hex.EncodeToString(reverseBytes(s)))
}
