//go:build !cuda
// +build !cuda

// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

/*
#cgo linux CFLAGS: -Irust/target/release -Irust-target-release-zk
#cgo linux LDFLAGS: -Lrust/target/release -Lrust-target-release-zk -l:libillium_zk.a -ldl -lpthread -lgcc_s -lc -lm -lssl -lcrypto -lstdc++
#cgo windows CFLAGS: -Irust/target/release
#cgo windows LDFLAGS: -Lrust/target/release -l:libillium_zk.lib
#cgo darwin CFLAGS: -Irust/target/release -Irust/target/aarch64-apple-darwin
#cgo darwin LDFLAGS: -Lrust/target/release -Irust/target/aarch64-apple-darwin -l:libillium_zk.a -lc++ -lssl -lcrypto -framework SystemConfiguration
*/
import "C"
