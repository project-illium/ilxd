// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSigValidator(t *testing.T) {
	sigCache := NewSigCache(10)
	sigValidator := NewSigValidator(sigCache)

	sk, pk, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	pkBytes, err := crypto.MarshalPublicKey(pk)
	assert.NoError(t, err)

	validatorID, err := peer.IDFromPublicKey(pk)
	assert.NoError(t, err)
	validatorIDBytes, err := validatorID.Marshal()
	assert.NoError(t, err)

	coinbaseTx := &transactions.CoinbaseTransaction{
		Validator_ID: validatorIDBytes,
	}
	sigHash, err := coinbaseTx.SigHash()
	assert.NoError(t, err)
	sig, err := sk.Sign(sigHash)
	assert.NoError(t, err)
	coinbaseTx.Signature = sig

	stakeTx := &transactions.StakeTransaction{
		Validator_ID: validatorIDBytes,
	}
	sigHash, err = stakeTx.SigHash()
	assert.NoError(t, err)
	sig, err = sk.Sign(sigHash)
	assert.NoError(t, err)
	stakeTx.Signature = sig

	mintTx := &transactions.MintTransaction{
		MintKey: pkBytes,
	}
	sigHash, err = mintTx.SigHash()
	assert.NoError(t, err)
	sig, err = sk.Sign(sigHash)
	assert.NoError(t, err)
	mintTx.Signature = sig

	err = sigValidator.Validate([]*transactions.Transaction{
		transactions.WrapTransaction(coinbaseTx),
		transactions.WrapTransaction(stakeTx),
		transactions.WrapTransaction(mintTx),
		transactions.WrapTransaction(&transactions.StandardTransaction{}),
	})
	assert.NoError(t, err)

	err = ValidateTransactionSig(transactions.WrapTransaction(coinbaseTx), sigCache)
	assert.NoError(t, err)

	coinbaseTx.Validator_ID = nil
	err = ValidateTransactionSig(transactions.WrapTransaction(coinbaseTx), NewSigCache(10))
	assert.Error(t, err)

	coinbaseTx.Validator_ID = validatorIDBytes
	coinbaseTx.Signature = nil
	err = ValidateTransactionSig(transactions.WrapTransaction(coinbaseTx), NewSigCache(10))
	assert.Error(t, err)
}
