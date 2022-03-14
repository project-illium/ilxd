// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/consensus"
	"path"
	"strings"

	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/repo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	black color = iota + 30
	red
	green
	yellow
	blue
	magenta
	cyan
	white
)

// color represents a text color.
type color uint8

// Add adds the coloring to the given string.
func (c color) Add(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}

var LogLevelMap = map[string]zapcore.Level{
	"debug":     zap.DebugLevel,
	"info":      zap.InfoLevel,
	"warning":   zap.WarnLevel,
	"error":     zap.ErrorLevel,
	"alert":     zap.DPanicLevel,
	"critical":  zap.PanicLevel,
	"emergency": zap.FatalLevel,
}

var logLevelSeverity = map[zapcore.Level]string{
	zapcore.DebugLevel:  "DEBUG",
	zapcore.InfoLevel:   "INFO",
	zapcore.WarnLevel:   "WARNING",
	zapcore.ErrorLevel:  "ERROR",
	zapcore.DPanicLevel: "CRITICAL",
	zapcore.PanicLevel:  "ALERT",
	zapcore.FatalLevel:  "EMERGENCY",
}

func setupLogging(logDir, level string, testnet bool) error {
	var cfg zap.Config
	if testnet {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	logLevel, ok := LogLevelMap[strings.ToLower(level)]
	if !ok {
		return errors.New("invalid log level")
	}
	cfg.Encoding = "console"
	cfg.Level = zap.NewAtomicLevelAt(logLevel)

	levelToColor := map[zapcore.Level]color{
		zapcore.DebugLevel:  magenta,
		zapcore.InfoLevel:   blue,
		zapcore.WarnLevel:   yellow,
		zapcore.ErrorLevel:  red,
		zapcore.DPanicLevel: red,
		zapcore.PanicLevel:  red,
		zapcore.FatalLevel:  red,
	}
	customLevelEncoder := func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + levelToColor[level].Add(logLevelSeverity[level]) + "]")
	}
	cfg.EncoderConfig.EncodeLevel = customLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.DisableCaller = true
	cfg.EncoderConfig.ConsoleSeparator = "  "

	var (
		logger *zap.Logger
		err    error
	)
	if logDir != "" {
		logRotator := &lumberjack.Logger{
			Filename:   path.Join(logDir, repo.DefaultLogFilename),
			MaxSize:    10, // Megabytes
			MaxBackups: 3,
			MaxAge:     30, // Days
		}

		lumberjackZapHook := func(e zapcore.Entry) error {
			logRotator.Write([]byte(fmt.Sprintf("%+v\n", e)))
			return nil
		}

		logger, err = cfg.Build(zap.Hooks(lumberjackZapHook))
		if err != nil {
			return err
		}
	} else {
		logger, err = cfg.Build()
		if err != nil {
			return err
		}
	}
	zap.ReplaceGlobals(logger)

	log = zap.S()
	repo.UpdateLogger()
	net.UpdateLogger()
	blockchain.UpdateLogger()
	consensus.UpdateLogger()
	return nil
}
