// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

import (
	"github.com/project-illium/logger"
	"github.com/pterm/pterm"
)

var log = logger.DisabledLogger.WithLevel(pterm.LogLevelDisabled)

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger *logger.Logger) {
	log = logger
}
