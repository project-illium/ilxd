// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package timelockedmultisig

import (
	"encoding/binary"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/project-illium/ilxd/zk/scripts/multisig"
	"time"
)

type PrivateParams struct {
	Signatures  [][]byte
	SigBitField uint8
}

func TimelockedMultisigScript(privateParams, publicParams interface{}) bool {
	priv, ok := privateParams.(*PrivateParams)
	if !ok {
		return false
	}
	pub, ok := publicParams.(*standard.UnlockingScriptInputs)
	if !ok {
		return false
	}
	if len(pub.ScriptParams) < 3 {
		return false
	}
	if len(pub.ScriptParams[0]) != 1 || len(pub.ScriptParams[1]) != 8 {
		return false
	}

	lockUntil := int64(binary.BigEndian.Uint64(pub.ScriptParams[1]))
	if pub.PublicParams.Locktime.Before(time.Unix(lockUntil, 0)) {
		return false
	}

	pubkeys := make([]crypto.PubKey, len(pub.ScriptParams)-1)
	for i := 1; i < len(pub.ScriptParams); i++ {
		key, err := crypto.UnmarshalPublicKey(pub.ScriptParams[i])
		if err != nil {
			return false
		}
		pubkeys[i-1] = key
	}

	valid, err := multisig.ValidateMultiSignature(pub.ScriptParams[0][0], pubkeys, priv.Signatures, priv.SigBitField, pub.PublicParams.SigHash)
	if err != nil || !valid {
		return false
	}
	return true
}
