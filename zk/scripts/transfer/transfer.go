// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package transfer

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/zk/circuits/smart"
)

type PrivateParams struct {
	Signature []byte
}

func TransferScript(privateParams, publicParams interface{}) bool {
	priv, ok := privateParams.(*PrivateParams)
	if !ok {
		return false
	}
	pub, ok := publicParams.(*smart.UnlockingSnarkParams)
	if !ok {
		return false
	}

	if len(pub.UserParams) != 1 {
		return false
	}

	pubkey, err := crypto.UnmarshalPublicKey(pub.UserParams[0])
	if err != nil {
		return false
	}

	valid, err := pubkey.Verify(pub.PublicParams.SigHash, priv.Signature)
	if err != nil || !valid {
		return false
	}
	return true
}
