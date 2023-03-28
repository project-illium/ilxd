// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package sync

import (
	"fmt"
	"github.com/project-illium/ilxd/blockchain"
	"testing"
)

func TestChainService_GetBlockTxids(t *testing.T) {
	err := error(blockchain.RuleError{
		ErrorCode:   2,
		Description: "adfasf",
	})

	_, ok := err.(blockchain.RuleError)
	fmt.Println(ok)
}
