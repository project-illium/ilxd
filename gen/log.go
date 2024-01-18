// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"github.com/pterm/pterm"
)

var log = pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger *pterm.Logger) {
	log = logger
}
