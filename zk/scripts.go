// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"embed"
	"github.com/project-illium/ilxd/zk/lurk/macros"
)

//go:embed lurk/basic_transfer.lurk
var basicTransferScriptLurk embed.FS
var basicTransferScriptData string

//go:embed lurk/password_script.lurk
var passwordScriptLurk embed.FS
var passwordScriptData string

func init() {
	mp, err := macros.NewMacroPreprocessor(macros.WithStandardLib(), macros.RemoveComments())
	if err != nil {
		panic(err)
	}
	data, err := basicTransferScriptLurk.ReadFile("lurk/basic_transfer.lurk")
	if err != nil {
		panic(err)
	}
	basicTransferScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}

	data, err = passwordScriptLurk.ReadFile("lurk/password_script.lurk")
	if err != nil {
		panic(err)
	}
	passwordScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}
}

// BasicTransferScript returns the basic transfer lurk script
func BasicTransferScript() string {
	return basicTransferScriptData
}

// PasswordScript returns the password lurk script
func PasswordScript() string {
	return passwordScriptData
}
