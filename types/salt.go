// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/zk"
	"math/big"
)

// RandomSalt generates a random number that is less than the
// lurk max field element.
func RandomSalt() ([32]byte, error) {
	upperBound := new(big.Int)
	upperBound.SetString(zk.LurkMaxFieldElement, 16)

	// Generate a random number in the range [0, upperBound)
	randomNum, err := rand.Int(rand.Reader, upperBound)
	if err != nil {
		return [32]byte{}, err
	}

	var ret [32]byte
	randomBytes := randomNum.Bytes()

	startIndex := len(ret) - len(randomBytes)
	copy(ret[startIndex:], randomBytes)
	return ret, nil
}
