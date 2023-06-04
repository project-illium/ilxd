// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"go.uber.org/zap/zapcore"
)

type ChainServer interface {
	BroadcastTxFunc(tx *transactions.Transaction) error
	SetLogLevelFunc(level zapcore.Level)
	ReindexChainFunc() error
	RequestBlockFunc(blockID types.ID, remotePeer peer.ID)
}
