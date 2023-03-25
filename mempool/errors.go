// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"errors"
	"fmt"
	"github.com/project-illium/ilxd/blockchain"
)

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
type AssertError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

// ruleError creates an RuleError given a set of arguments.
func ruleError(c blockchain.ErrorCode, desc string) blockchain.RuleError {
	return blockchain.RuleError{ErrorCode: c, Description: desc}
}

type ErrorCode int

const (
	ErrFeeTooLow = iota
	ErrMinStake
	ErrDuplicateCoinbase
	ErrTreasuryWhitelist
)

var (
	ErrDuplicateTx = errors.New("tx already in mempool")
	ErrNotFound    = errors.New("tx not found in pool")
)

// Map of ErrorCode values back to their constant names for pretty printing.
var errorCodeStrings = map[ErrorCode]string{
	ErrFeeTooLow:         "ErrFeeTooLow",
	ErrMinStake:          "ErrMinStake",
	ErrDuplicateCoinbase: "ErrDuplicateCoinbase",
	ErrTreasuryWhitelist: "ErrTreasuryWhitelist",
}

// String returns the ErrorCode as a human-readable name.
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

// PolicyError identifies a mempool policy violation.  It is used to
// indicate that the transaction met the consensus validity rules but
// the node chose not to accept it because it violated its own policy
// rule
type PolicyError struct {
	ErrorCode   ErrorCode // Describes the kind of error
	Description string    // Human-readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e PolicyError) Error() string {
	return e.Description
}

// policyError creates an RuleError given a set of arguments.
func policyError(c ErrorCode, desc string) PolicyError {
	return PolicyError{ErrorCode: c, Description: desc}
}
