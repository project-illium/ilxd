// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/indexers"
	"github.com/project-illium/ilxd/consensus"
	"github.com/project-illium/ilxd/gen"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/policy/protocol"
	"github.com/project-illium/ilxd/repo/blockstore"
	"github.com/project-illium/ilxd/sync"
	"github.com/project-illium/walletlib"
	"github.com/pterm/pterm"
	"path"
	"strings"

	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/repo"
	"gopkg.in/natefinch/lumberjack.v2"
)

var LogLevelMap = map[string]pterm.LogLevel{
	"trace":   pterm.LogLevelTrace,
	"debug":   pterm.LogLevelDebug,
	"info":    pterm.LogLevelInfo,
	"warning": pterm.LogLevelWarn,
	"error":   pterm.LogLevelError,
	"fatal":   pterm.LogLevelFatal,
}

func setupLogging(logDir, level string, testnet bool) error {
	logLevel, ok := LogLevelMap[strings.ToLower(level)]
	if !ok {
		return errors.New("invalid log level")
	}

	log = log.WithCustomLogger(pterm.DefaultLogger.WithLevel(logLevel))

	if logDir != "" {
		logRotator := &lumberjack.Logger{
			Filename:   path.Join(logDir, repo.DefaultLogFilename),
			MaxSize:    10, // Megabytes
			MaxAge:     30, // Days
			MaxBackups: 3,
		}
		log = log.WithCustomLogger(pterm.DefaultLogger.
			WithLevel(logLevel).
			WithFormatter(pterm.LogFormatterJSON).
			WithWriter(logRotator))
	}

	repo.UseLogger(log)
	net.UseLogger(log)
	blockchain.UseLogger(log)
	consensus.UseLogger(log)
	gen.UseLogger(log)
	sync.UseLogger(log)
	mempool.UseLogger(log)
	walletlib.UseLogger(log)
	indexers.UseLogger(log)
	policy.UseLogger(log)
	protocol.UseLogger(log)
	blockstore.UseLogger(log)
	return nil
}

func UpdateLogLevel(level string) error {
	logLevel, ok := LogLevelMap[strings.ToLower(level)]
	if !ok {
		return errors.New("invalid log level")
	}
	log = log.WithLevel(logLevel)
	return nil
}
