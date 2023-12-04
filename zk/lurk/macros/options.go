// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros

import (
	"embed"
	"io/fs"
	"os"
)

// Option is configuration option function for the MacroPreprocessor
type Option func(cfg *config) error

type fsDirectory struct {
	fileSystem fs.FS
	path       string
}

// DependencyDir sets the dependency directory that is used to look
// up imported modules.
func DependencyDir(depDir string) Option {
	return func(cfg *config) error {
		cfg.depDir = &fsDirectory{
			fileSystem: os.DirFS(depDir),
			path:       ".",
		}
		return nil
	}
}

//go:embed deps/std/*
var embeddedDependencyDir embed.FS

// WithStandardLib creates an embedded dependency directory
// containing only the standard library. This is not compatible
// with DependencyDir.
func WithStandardLib() Option {
	return func(cfg *config) error {
		cfg.depDir = &fsDirectory{
			fileSystem: embeddedDependencyDir,
			path:       "deps",
		}
		return nil
	}
}

func RemoveComments() Option {
	return func(cfg *config) error {
		cfg.removeComments = true
		return nil
	}
}

type config struct {
	depDir         *fsDirectory
	removeComments bool
}
