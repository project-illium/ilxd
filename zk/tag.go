// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"errors"
)

var (
	OutputTrue  = []byte{4, 200, 219, 197, 70, 43, 38, 162, 69, 194, 112, 0, 19, 99, 1, 240, 70, 225, 92, 182, 50, 158, 201, 58, 139, 58, 93, 133, 73, 161, 152, 86}
	OutputFalse = []byte{44, 231, 13, 115, 220, 26, 132, 163, 216, 74, 115, 1, 101, 44, 104, 11, 34, 107, 142, 141, 165, 31, 169, 229, 24, 212, 54, 87, 140, 94, 52, 39}
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
