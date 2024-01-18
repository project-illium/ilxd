// Copyright (c) 2022 Project Illium
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
	"github.com/project-illium/ilxd/sync"
	"github.com/project-illium/walletlib"
	"github.com/pterm/pterm"
	"io"
	"os"
	"path"
	"strings"

	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/repo"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logWriter struct {
	fileLogger *lumberjack.Logger
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	if _, err := os.Stdout.Write(p); err != nil {
		return 0, err
	}
	return w.fileLogger.Write(p)
}

var LogLevelMap = map[string]pterm.LogLevel{
	"trace":   pterm.LogLevelTrace,
	"debug":   pterm.LogLevelDebug,
	"info":    pterm.LogLevelInfo,
	"warning": pterm.LogLevelWarn,
	"error":   pterm.LogLevelError,
	"fatal":   pterm.LogLevelFatal,
}

var logLevelSeverity = map[pterm.LogLevel]string{
	pterm.LogLevelTrace: "TRACE",
	pterm.LogLevelDebug: "DEBUG",
	pterm.LogLevelInfo:  "INFO",
	pterm.LogLevelWarn:  "WARNING",
	pterm.LogLevelError: "ERROR",
	pterm.LogLevelFatal: "FATAL",
}

func setupLogging(logDir, level string, testnet bool) error {
	logLevel, ok := LogLevelMap[strings.ToLower(level)]
	if !ok {
		return errors.New("invalid log level")
	}

	var writer io.Writer = os.Stdout

	if logDir != "" {
		logRotator := &lumberjack.Logger{
			Filename:   path.Join(logDir, repo.DefaultLogFilename),
			MaxSize:    10, // Megabytes
			MaxAge:     30, // Days
			MaxBackups: 3,
		}

		// logRotator.Write([]byte(fmt.Sprintf("%+v\n", e)))
		writer = &logWriter{fileLogger: logRotator}
	}

	log = pterm.DefaultLogger.WithWriter(writer).WithLevel(logLevel)

	repo.UseLogger(log)
	net.UseLogger(log)
	blockchain.UseLogger(log)
	consensus.UseLogger(log)
	gen.UseLogger(log)
	sync.UseLogger(log)
	mempool.UseLogger(log)
	walletlib.UseLogger(log)
	indexers.UseLogger(log)
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
