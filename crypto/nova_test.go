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

	message := []byte("message")
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
	raw, _ := pub3.Raw()
	fmt.Println(hex.EncodeToString(x), hex.EncodeToString(y), hex.EncodeToString(raw))
}
