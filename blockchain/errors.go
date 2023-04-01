// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
)

// OrphanBlockError that the processed block is an orphan.
type OrphanBlockError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e OrphanBlockError) Error() string {
	return "orphan block"
}

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
type AssertError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

type ErrorCode int

const (
	ErrDuplicateBlock = iota
	ErrInvalidProducer
	ErrDoesNotConnect
	ErrInvalidHeight
	ErrInvalidTimestamp
	ErrInvalidHeaderSignature
	ErrEmptyBlock
	ErrInvalidTxRoot
	ErrDoubleSpend
	ErrDuplicateCoinbase
	ErrBlockStakeSpend
	ErrInvalidTx
	ErrInvalidGenesis
	ErrUnknownTxEnum
	ErrBlockSort
	ErrRestakeTooEarly
)

// Map of ErrorCode values back to their constant names for pretty printing.
var errorCodeStrings = map[ErrorCode]string{
	ErrDuplicateBlock:         "ErrDuplicateBlock",
	ErrInvalidProducer:        "ErrInvalidProducer",
	ErrDoesNotConnect:         "ErrDoesNotConnect",
	ErrInvalidHeight:          "ErrInvalidHeight",
	ErrInvalidTimestamp:       "ErrInvalidTimestamp",
	ErrInvalidHeaderSignature: "ErrInvalidHeaderSignature",
	ErrEmptyBlock:             "ErrEmptyBlock",
	ErrInvalidTxRoot:          "ErrInvalidTxRoot",
	ErrDoubleSpend:            "ErrDoubleSpend",
	ErrDuplicateCoinbase:      "ErrDuplicateCoinbase",
	ErrBlockStakeSpend:        "ErrBlockStakeSpend",
	ErrInvalidTx:              "ErrInvalidTx",
	ErrInvalidGenesis:         "ErrInvalidGenesis",
	ErrUnknownTxEnum:          "ErrUnknownTxEnum",
	ErrBlockSort:              "ErrBlockSort",
	ErrRestakeTooEarly:        "ErrRestakeTooEarly",
}

// String returns the ErrorCode as a human-readable name.
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

// RuleError identifies a rule violation.  It is used to indicate that
// processing of a block or transaction failed due to one of the many validation
// rules.  The caller can use type assertions to determine if a failure was
// specifically due to a rule violation and access the ErrorCode field to
// ascertain the specific reason for the rule violation.
type RuleError struct {
	ErrorCode   ErrorCode // Describes the kind of error
	Description string    // Human-readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e RuleError) Error() string {
	return e.Description
}

// ruleError creates an RuleError given a set of arguments.
func ruleError(c ErrorCode, desc string) RuleError {
	return RuleError{ErrorCode: c, Description: desc}
}

func ErrorIs(err error, code ErrorCode) bool {
	if ruleError, ok := err.(RuleError); ok && ruleError.ErrorCode == code {
		return true
	}
	return false
}
