// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package multisig

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"math/bits"
)

type PrivateParams struct {
	Signatures  [][]byte
	SigBitField uint8
}

func MultisigScript(privateParams, publicParams interface{}) bool {
	priv, ok := privateParams.(*PrivateParams)
	if !ok {
		return false
	}
	pub, ok := publicParams.(*standard.UnlockingSnarkParams)
	if !ok {
		return false
	}
	if len(pub.UserParams) < 2 {
		return false
	}
	if len(pub.UserParams[0]) != 1 {
		return false
	}

	pubkeys := make([]crypto.PubKey, len(pub.UserParams)-1)
	for i := 1; i < len(pub.UserParams); i++ {
		key, err := crypto.UnmarshalPublicKey(pub.UserParams[i])
		if err != nil {
			return false
		}
		pubkeys[i-1] = key
	}

	valid, err := ValidateMultiSignature(pub.UserParams[0][0], pubkeys, priv.Signatures, priv.SigBitField, pub.PublicParams.SigHash)
	if err != nil || !valid {
		return false
	}
	return true
}

func ValidateMultiSignature(threshold uint8, pubkeys []crypto.PubKey, signatures [][]byte, sigBitField uint8, sigHash []byte) (bool, error) {
	if len(signatures) > 8 || uint8(len(signatures)) < threshold {
		return false, nil
	}
	if bits.OnesCount8(sigBitField) != len(signatures) {
		return false, nil
	}
	if len(pubkeys) > 8 {
		return false, nil
	}
	sigIndex := 0
	for i := 0; i < len(pubkeys); i++ {
		f := uint8(1 << i)
		if f&sigBitField >= 1 {
			valid, err := pubkeys[i].Verify(sigHash, signatures[sigIndex])
			if !valid || err != nil {
				return false, err
			}
			sigIndex++
		}
	}

	return true, nil
}
