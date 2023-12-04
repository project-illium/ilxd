// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestBasicTransferScript(t *testing.T) {
	lurkProgram := MultisigScript()

	//lurkProgram = strings.ReplaceAll(lurkProgram, "\n", "")
	//lurkProgram = strings.ReplaceAll(lurkProgram, "\t", "")
	//lurkProgram = strings.Join(strings.Fields(lurkProgram), " ")
	comm, err := LurkCommit(lurkProgram)
	assert.NoError(t, err)

	fmt.Println(lurkProgram)
	fmt.Println(hex.EncodeToString(comm))
}

func TestMakeMultisigUnlockingParams(t *testing.T) {
	priv1, pub1, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	priv2, pub2, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	_, pub3, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	sigHash := make([]byte, 32)
	rand.Read(sigHash)

	sig1, err := priv1.Sign(sigHash)
	assert.NoError(t, err)
	sig2, err := priv2.Sign(sigHash)
	assert.NoError(t, err)

	pubkeys := make([][]byte, 0, 6)
	x1, y1 := pub1.(*icrypto.NovaPublicKey).ToXY()
	x2, y2 := pub2.(*icrypto.NovaPublicKey).ToXY()
	x3, y3 := pub3.(*icrypto.NovaPublicKey).ToXY()
	pubkeys = append(pubkeys, x1, y1, x2, y2, x3, y3)

	script, err := MakeMultisigUnlockingParams(pubkeys, [][]byte{sig1, sig2}, sigHash)
	assert.NoError(t, err)

	re := regexp.MustCompile(`0x[0-9a-fA-F]+`)
	expected := `(cons (cons 1 (cons 1 (cons 0 nil))) (cons (cons 0xe4f41e9e9c51a86e127a13af323ae286ed43d1df574b468d23c4216bceac0396 0xb38a1df6b53c293dfe51474edaca38af6636e4f351586656ab9c8409cfac4f36) (cons (cons 0xb5bbac5280a1c2d6b0b89d43fdea193d73e3be95ddc25d6a1b21b114aba50d11 0xce6dccc121b5572a4599224cf7cf228f37a2a1e56267f1cb9e3bd317cfb45226) nil)))`
	assert.Equal(t, re.ReplaceAllString(expected, ""), re.ReplaceAllString(string(script), ""))
}
