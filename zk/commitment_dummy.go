//go:build skiprusttests
// +build skiprusttests

// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
)

func LurkCommit(expr string) (types.ID, error) {
	return hash.HashFunc([]byte(expr))
}
