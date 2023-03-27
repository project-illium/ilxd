// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"fmt"
	"github.com/project-illium/ilxd/params/hash"
)

var ErrIDStrSize = fmt.Errorf("max ID string length is %v bytes", hash.HashSize*2)

type ID [hash.HashSize]byte

// Compare returns 1 if hash > target, -1 if hash < target and
// 0 if hash == target.
func (id ID) Compare(target ID) int {
	for i := 0; i < len(id); i++ {
		a := id[i]
		b := target[i]
		if a > b {
			return 1
		}
		if a < b {
			return -1
		}
	}
	return 0
}

func (id ID) String() string {
	return hex.EncodeToString(id[:])
}

func (id ID) Bytes() []byte {
	return id[:]
}

func (id *ID) SetBytes(data []byte) {
	copy(id[:], data)
}

func (id *ID) MarshalJSON() ([]byte, error) {
	return []byte(hex.EncodeToString(id[:])), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	i, err := NewIDFromString(string(data))
	if err != nil {
		return err
	}
	id = &i
	return nil
}

func NewID(digest []byte) ID {
	var sh ID
	sh.SetBytes(digest)
	return sh
}

func NewIDFromString(id string) (ID, error) {
	// Return error if hash string is too long.
	if len(id) > hash.HashSize*2 {
		return ID{}, ErrIDStrSize
	}
	ret, err := hex.DecodeString(id)
	if err != nil {
		return ID{}, err
	}
	var newID ID
	newID.SetBytes(ret)
	return newID, nil
}

func NewIDFromData(data []byte) ID {
	var id ID
	hash := hash.HashFunc(data)
	id.SetBytes(hash)
	return id
}
