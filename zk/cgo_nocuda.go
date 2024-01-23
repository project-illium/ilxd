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
#cgo darwin CFLAGS: -Irust/target/release -Irust/target/x86_64-apple-darwin/release
#cgo darwin LDFLAGS: -Lrust/target/release -Lrust/target/x86_64-apple-darwin/release -lillium_zk -lc++ -lssl -lcrypto -framework SystemConfiguration
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <fcntl.h>
extern int lurk_commit(const char* expr, unsigned char* out);
// Function to save the current stderr file descriptor and redirect stderr to /dev/null
int redirect_stderr() {
    int stderr_copy = dup(2);
    int dev_null = open("/dev/null", O_WRONLY);
    dup2(dev_null, 2);
    close(dev_null);
    return stderr_copy;
}
// Function to restore stderr from the saved file descriptor
void restore_stderr(int stderr_copy) {
    dup2(stderr_copy, 2);
    close(stderr_copy);
}
void load_public_params();
int create_proof_ffi(
    const char* lurk_program,
    const char* private_params,
    const char* public_params,
    uint8_t* proof,
    size_t* proof_len,
    uint8_t* output_tag,
    uint8_t* output_val);
int verify_proof_ffi(
    const char* lurk_program,
    const char* public_params,
    const uint8_t* proof,
    size_t proof_size,
    const uint8_t* expected_tag,
    const uint8_t* expected_output);
int eval_ffi(
    const char* lurk_program,
    const char* private_params,
    const char* public_params,
    uint8_t* output_tag,
    uint8_t* output_val,
	size_t* iterations);
*/
import "C"
