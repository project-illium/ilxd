// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSigValidator(t *testing.T) {
	sigCache := NewSigCache(10)
	sigValidator := NewSigValidator(sigCache)

	sk, pk, err := crypto.GenerateEd25519Key(rand.Reader)
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

	sk2, pk2, err := icrypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	pkBytes2, err := crypto.MarshalPublicKey(pk2)
	assert.NoError(t, err)

	mintTx := &transactions.MintTransaction{
		MintKey: pkBytes2,
	}
	sigHash, err = mintTx.SigHash()
	assert.NoError(t, err)
	sig, err = sk2.Sign(sigHash)
	assert.NoError(t, err)
	mintTx.Signature = sig

	err = sigValidator.Validate([]*transactions.Transaction{
		transactions.WrapTransaction(coinbaseTx),
		transactions.WrapTransaction(stakeTx),
		transactions.WrapTransaction(mintTx),
		transactions.WrapTransaction(&transactions.StandardTransaction{}),
	})
	assert.NoError(t, err)

	c := ValidateTransactionSig(transactions.WrapTransaction(coinbaseTx), sigCache)
	err = <-c
	assert.NoError(t, err)

	coinbaseTx.Validator_ID = nil
	c = ValidateTransactionSig(transactions.WrapTransaction(coinbaseTx), NewSigCache(10))
	err = <-c
	assert.Error(t, err)

	coinbaseTx.Validator_ID = validatorIDBytes
	coinbaseTx.Signature = nil
	c = ValidateTransactionSig(transactions.WrapTransaction(coinbaseTx), NewSigCache(10))
	err = <-c
	assert.Error(t, err)
}
