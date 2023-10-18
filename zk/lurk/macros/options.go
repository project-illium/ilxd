// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros

// Option is configuration option function for the MacroPreprocessor
type Option func(cfg *config) error

// DependencyDir sets the dependency directory that is used to look
// up imported modules.
func DependencyDir(depDir string) Option {
	return func(cfg *config) error {
		cfg.depDir = depDir
		return nil
	}
}

type config struct {
	depDir string
}
