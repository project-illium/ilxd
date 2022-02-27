// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import "go.uber.org/zap"

var log = zap.S()

func UpdateLogger() {
	log = zap.S()
}
