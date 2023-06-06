// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncryption(t *testing.T) {
	priv, pub, err := GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	message := []byte("message")
	//rand.Read(message)

	cipherText, err := Encrypt(pub, message)
	assert.NoError(t, err)

	plainText, err := Decrypt(priv, cipherText)
	assert.NoError(t, err)

	assert.Equal(t, message, plainText)
}
