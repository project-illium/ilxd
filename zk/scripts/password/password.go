// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package password

import (
	"bytes"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"golang.org/x/crypto/blake2b"
)

type PrivateParams struct {
	Password []byte
}

func PasswordScript(privateParams, publicParams interface{}) bool {
	priv, ok := privateParams.(*PrivateParams)
	if !ok {
		return false
	}
	pub, ok := publicParams.(*standard.UnlockingSnarkParams)
	if !ok {
		return false
	}

	if len(pub.UserParams) != 1 {
		return false
	}

	hash := pub.UserParams[0]
	calculatedHash := blake2b.Sum256(priv.Password)
	return bytes.Equal(hash, calculatedHash[:])
}
