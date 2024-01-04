// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"fmt"
	"github.com/project-illium/ilxd/types"
	"time"
)

type StandardPrivateParams struct {
	Inputs  []PrivateInput
	Outputs []PrivateOutput
}

func (priv *StandardPrivateParams) ToExpr() string {
	inputs := ""
	outputs := ""
	return "(cons " + inputs + " (cons " + outputs + " nil))"
}

/*
!(param nullifiers <index>)                           ;; expands to: (nth <index> (nth 0 public-params))
!(param txo-root)                                     ;; expands to: (nth 1 public-params)
!(param fee)                                          ;; expands to: (nth 2 public-params)
!(param coinbase)                                     ;; expands to: (nth 3 public-params)
!(param mint-id)                                      ;; expands to: (nth 4 public-params)
!(param mint-amount)                                  ;; expands to: (nth 5 public-params)
!(param pub-out <index>)                              ;; expands to: (nth <index> (nth 6 public-params))
!(param sighash)                                      ;; expands to: (nth 7 public-params)
!(param locktime)                                     ;; expands to: (nth 8 public-params)
!(param locktime-precision)                           ;; expands to: (nth 9 public-params)
*/

type StandardPublicParams struct {
	Nullifiers        []types.Nullifier
	TXORoot           types.ID
	Fee               types.Amount
	Coinbase          types.Amount
	MintID            types.ID
	MintAmount        types.Amount
	Outputs           []PublicOutput
	SigHash           types.ID
	Locktime          time.Time
	LocktimePrecision time.Duration
}

func (pub *StandardPublicParams) ToExpr() string {
	nullifiers := ""
	for _, n := range pub.Nullifiers {
		nullifiers += fmt.Sprintf("(cons 0x%x ", n.Bytes())
	}
	nullifiers += "nil)"
	for i := 0; i < len(pub.Nullifiers)-1; i++ {
		nullifiers += ")"
	}

	outputs := ""
	for _, o := range pub.Outputs {
		output := fmt.Sprintf("(cons 0x%x ", o.Commitment.Bytes())

		const chunkSize = 32

		ciphertext := ""
		nChunks := 0
		for i := 0; i < len(o.CipherText); i += chunkSize {
			nChunks++
			end := i + chunkSize
			if end > len(o.CipherText) {
				end = len(o.CipherText)
			}

			chunk := make([]byte, end-i)
			copy(chunk, o.CipherText[i:end])

			// Lurk elements exist within a finite field and cannot
			// exceed the maximum field element. Here we set the two
			// most significant bits of each ciphertext chunk to zero
			// to fit within the max size.
			//
			// In the normal case where the ciphertext is curve25519
			// this doesn't matter because we can't compute that inside
			// the circuit anyway. But if you have a use case where you
			// validate the ciphertext field in any way you need to take
			// this into account.
			if len(chunk) == chunkSize {
				chunk[0] &= 0x3f
			}

			ciphertext += fmt.Sprintf("(cons 0x%x ", chunk)
		}
		ciphertext += "nil)"
		for i := 0; i < nChunks-1; i++ {
			ciphertext += ")"
		}
		output += "(cons " + ciphertext + " nil))"
		outputs += "(cons " + output + " "
	}
	outputs += "nil)"
	for i := 0; i < len(pub.Outputs)-1; i++ {
		outputs += ")"
	}

	expr := "(cons " + nullifiers +
		fmt.Sprintf(" (cons 0x%x ", pub.TXORoot.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.Fee) +
		fmt.Sprintf("(cons %d ", pub.Coinbase) +
		fmt.Sprintf(" (cons 0x%x ", pub.MintID.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.MintAmount) +
		"(cons " + outputs +
		fmt.Sprintf(" (cons 0x%x ", pub.SigHash.Bytes()) +
		fmt.Sprintf("(cons %d ", pub.Locktime.Unix()) +
		fmt.Sprintf("(cons %d ", int64(pub.LocktimePrecision.Seconds())) +
		"nil))))))))))"

	return expr
}
