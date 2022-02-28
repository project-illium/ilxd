// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package wallet

import (
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/params"
	"testing"
)

func TestBasicAddress(t *testing.T) {
	_, pubkey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	_, viewKey, err := GenerateCurve25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	ss := SpendScript{
		Threshold: 1,
		Pubkeys:   []crypto.PubKey{pubkey},
	}

	addr, err := NewBasicAddress(ss, viewKey.(*Curve25519PublicKey), &params.MainnetParams)
	if err != nil {
		t.Fatal(err)
	}

	addr2, err := DecodeAddress(addr.String(), &params.MainnetParams)
	if err != nil {
		t.Fatal(err)
	}

	if addr2.String() != addr.String() {
		t.Error("Decoded address does not match encoded")
	}

	fmt.Println(addr2.String())
}
