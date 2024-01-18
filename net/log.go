// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"github.com/pterm/pterm"
)

var log = pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger *pterm.Logger) {
	log = logger
}
