//go:build !skiprusttests
// +build !skiprusttests

// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

/*
#cgo linux CFLAGS: -Irust/target/release -Irust-target-release-zk
#cgo linux LDFLAGS: -Lrust/target/release -Lrust-target-release-zk -l:libillium_zk.a -ldl -lpthread -lgcc_s -lc -lm -lssl -lcrypto -lstdc++ -L/usr/local/cuda/lib64 -L/usr/local/cuda/targets/x86_64-linux/lib -lcudart
#cgo windows CFLAGS: -Irust/target/release
#cgo windows LDFLAGS: -Lrust/target/release -l:libillium_zk.lib
#cgo darwin CFLAGS: -Irust/target/release -Irust/target/x86_64-apple-darwin/release
#cgo darwin LDFLAGS: -Lrust/target/release -Lrust/target/x86_64-apple-darwin/release -lillium_zk -lc++ -lssl -lcrypto -framework SystemConfiguration
#include <stdlib.h>
void load_public_params();
*/
import "C"
import (
	"sync"
)

var once sync.Once

// LoadZKPublicParameters loads the lurk public parameters from disk
// into memory or generates them if this is the first startup.
func LoadZKPublicParameters() {
	log.Info("Loading lurk public parameters...")
	once.Do(func() {
		C.load_public_params()
	})
}
