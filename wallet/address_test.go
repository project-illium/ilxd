// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package wallet

import (
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBasicAddress(t *testing.T) {
	_, pubkey, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	_, viewKey, err := GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	_, verificationKey, err := GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	verificationKeyRaw, err := verificationKey.Raw()
	assert.NoError(t, err)
	pubKeyRaw, err := pubkey.Raw()
	assert.NoError(t, err)

	us := types.UnlockingScript{
		SnarkVerificationKey: verificationKeyRaw,
		PublicParams:         [][]byte{pubKeyRaw},
	}

	addr, err := NewBasicAddress(us, viewKey, &params.MainnetParams)
	assert.NoError(t, err)

	addr2, err := DecodeAddress(addr.String(), &params.MainnetParams)
	assert.NoError(t, err)

	if addr2.String() != addr.String() {
		t.Error("Decoded address does not match encoded")
	}

	fmt.Println(addr2.String())
}
