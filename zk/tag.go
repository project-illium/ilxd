// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"errors"
)

var (
	OutputTrue  = []byte{31, 92, 240, 53, 64, 81, 52, 184, 242, 164, 206, 155, 203, 246, 62, 58, 99, 64, 80, 9, 206, 22, 104, 13, 186, 172, 246, 192, 146, 161, 140, 76}
	OutputFalse = []byte{29, 85, 200, 194, 103, 218, 100, 115, 6, 158, 39, 183, 243, 138, 4, 149, 54, 111, 248, 230, 51, 5, 165, 229, 248, 91, 9, 187, 12, 36, 47, 3}
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
