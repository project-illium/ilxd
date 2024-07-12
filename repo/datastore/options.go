// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package datastore

import (
	"github.com/project-illium/ilxd/params"
)

// Option is configuration option function for the Datastore
type Option func(cfg *config) error

// WithNoBlockstore disables the blockstore for the database
func WithNoBlockstore() Option {
	return func(cfg *config) error {
		cfg.noBlockstore = true
		return nil
	}
}

// WithParams sets the params for the datastore
func WithParams(params *params.NetworkParams) Option {
	return func(cfg *config) error {
		cfg.params = params
		return nil
	}
}

type config struct {
	params       *params.NetworkParams
	noBlockstore bool
}
