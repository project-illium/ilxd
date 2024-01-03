// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"errors"
)

// Tag represents a Lurk data type
type Tag uint8

const (
	TagNil Tag = iota
	TagCons
	TagSym
	TagFun
	TagNum
	TagThunk
	TagStr
	TagChar
	TagComm
	TagU64
	TagKey
	TagCproc
)

// TagFromBytes returns a tag from a big endian byte slice
func TagFromBytes(b []byte) (Tag, error) {
	if len(b) < 1 {
		return TagNil, errors.New("slice is not large enough")
	}
	return Tag(b[len(b)-1]), nil
}
