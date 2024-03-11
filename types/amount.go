// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

const NanosPerILX = 1e9

// Amount represents the base illium monetary unit (nanoillium or nanos).
// The total number of nanoillium issued is not expected to overflow an uint64
// for 250 years from genesis.
type Amount uint64

// AmountFromILX returns an amount (in nanoillium) from an ILX float string
func AmountFromILX(floatString string) (Amount, error) {
	if strings.TrimSpace(floatString) == "" {
		return 0, nil
	}
	amt, err := parseFormattedAmount(floatString)
	if err != nil {
		return Amount(0), err
	}
	return Amount(amt), nil
}

// ToBytes returns the byte representation of the amount
func (a Amount) ToBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(a))
	return b
}

// ToILX returns the amount, formatted as ILX
func (a Amount) ToILX() float64 {
	str := formatAmountString(uint64(a))
	f, _ := strconv.ParseFloat(str, 64)
	return f
}

// MarshalJSON returns the amount in ILX as a float string
func (a *Amount) MarshalJSON() ([]byte, error) {
	return []byte(formatAmountString(uint64(*a))), nil
}

// UnmarshalJSON unmarshals an ILX float string to an Amount.
func (a *Amount) UnmarshalJSON(data []byte) error {
	str := string(data)

	amt, err := parseFormattedAmount(str)
	if err != nil {
		return err
	}
	*a = Amount(amt)
	return nil
}

func formatAmountString(num uint64) string {
	// Convert the number to a string.
	numStr := fmt.Sprintf("%d", num)

	// Ensure the string is at least 9 characters long by prepending zeros.
	for len(numStr) < 9 {
		numStr = "0" + numStr
	}

	// Insert the decimal point 9 characters from the end.
	numStr = numStr[:len(numStr)-9] + "." + numStr[len(numStr)-9:]

	// Drop trailing zeros and the decimal point if necessary.
	numStr = strings.TrimRight(numStr, "0")
	numStr = strings.TrimRight(numStr, ".")

	// If there's no number before the decimal, add a zero.
	if numStr == "" || numStr[0] == '.' {
		numStr = "0" + numStr
	}

	return numStr
}

func parseFormattedAmount(formattedStr string) (uint64, error) {
	// Remove the decimal point to handle the number as a whole.
	noDecimalStr := strings.Replace(formattedStr, ".", "", -1)

	// Add trailing zeros if necessary to ensure the length is at least 9.
	// This accounts for cases where the original number had trailing zeros that were dropped.
	dotIndex := strings.Index(formattedStr, ".")
	missingZeros := 9 - (len(formattedStr) - dotIndex - 1)
	if dotIndex == -1 {
		missingZeros = 9
	}
	for i := 0; i < missingZeros; i++ {
		noDecimalStr += "0"
	}

	// Convert back to uint64.
	num, err := strconv.ParseUint(noDecimalStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return num, nil
}
