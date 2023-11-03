// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package merkledb

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/project-illium/ilxd/params/hash"
)

type HashVal []byte

func NewHashVal() HashVal {
	return make(HashVal, 32)
}

func (h HashVal) IsNil() bool {
	return bytes.Equal(h, make([]byte, 32)) || bytes.Equal(h, hash.HashMerkleBranches(make([]byte, 32), make([]byte, 32)))
}

func (h HashVal) IsData() bool {
	return len(h) == 64
}

func (h HashVal) Key() []byte {
	k := make([]byte, 32)
	copy(k, h[:32])
	return k
}

func (h HashVal) ValueHash() ([]byte, error) {
	if len(h) != 64 {
		return nil, errors.New("hashval is not data")
	}
	vh := make([]byte, 32)
	copy(vh, h[32:])
	return vh, nil
}

func (h HashVal) String() string {
	return hex.EncodeToString(h)
}

func (h HashVal) Bytes() []byte {
	return h.Clone()
}

func (h HashVal) Clone() HashVal {
	h2 := make(HashVal, len(h))
	copy(h2, h)
	return h2
}
