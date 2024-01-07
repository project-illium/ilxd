// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk_test

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	zk.LoadZKPublicParameters()
	os.Exit(m.Run())
}

func TestProve(t *testing.T) {
	r, err := zk.RandomFieldElement()
	assert.NoError(t, err)
	h, err := zk.LurkCommit(fmt.Sprintf("0x%x", r))
	assert.NoError(t, err)

	program := "(lambda (priv pub) (= (num (commit priv)) pub))"

	proof, err := zk.Prove(program, zk.Expr(fmt.Sprintf("0x%x", r)), zk.Expr(fmt.Sprintf("0x%x", h)))
	assert.NoError(t, err)

	valid, err := zk.Verify(program, zk.Expr(fmt.Sprintf("0x%x", h)), proof)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestCoprocessors(t *testing.T) {
	zk.LoadZKPublicParameters()

	t.Run("and", func(t *testing.T) {
		program := `(lambda (priv pub) (letrec ((and (lambda (a b)
                                    (eval (cons 'coproc_and (cons a (cons b nil)))))))
                            (= (and priv pub) 4)))`

		proof, err := zk.Prove(program, zk.Expr("7"), zk.Expr("12"))
		assert.NoError(t, err)

		valid, err := zk.Verify(program, zk.Expr("12"), proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("or", func(t *testing.T) {
		program := `(lambda (priv pub) (letrec ((or (lambda (a b)
                                    (eval (cons 'coproc_or (cons a (cons b nil)))))))
                            (= (or priv pub) 31)))`

		proof, err := zk.Prove(program, zk.Expr("19"), zk.Expr("15"))
		assert.NoError(t, err)

		valid, err := zk.Verify(program, zk.Expr("15"), proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("xor", func(t *testing.T) {
		program := `(lambda (priv pub) (letrec ((xor (lambda (a b)
                                    (eval (cons 'coproc_xor (cons a (cons b nil)))))))
                            (= (xor priv pub) 6)))`

		proof, err := zk.Prove(program, zk.Expr("3"), zk.Expr("5"))
		assert.NoError(t, err)

		valid, err := zk.Verify(program, zk.Expr("5"), proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("sha256", func(t *testing.T) {
		r, err := zk.RandomFieldElement()
		assert.NoError(t, err)

		// Lurk variables must fit within the finite field.
		// As such the hash output has the two most significant
		// bits set to zero.
		h := sha256.Sum256(r[:])
		h[0] &= 0b00111111

		program := `(lambda (priv pub) (letrec ((sha256 (lambda (preimage)
                                    (eval (cons 'coproc_sha256 (cons preimage nil))))))
                            (= (sha256 priv) pub)))`

		proof, err := zk.Prove(program, zk.Expr(fmt.Sprintf("0x%x", r)), zk.Expr(fmt.Sprintf("0x%x", h)))
		assert.NoError(t, err)

		valid, err := zk.Verify(program, zk.Expr(fmt.Sprintf("0x%x", h)), proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("blake2s", func(t *testing.T) {
		left, err := zk.RandomFieldElement()
		assert.NoError(t, err)
		right, err := zk.RandomFieldElement()
		assert.NoError(t, err)

		h := hash.HashMerkleBranches(left[:], right[:])

		program := fmt.Sprintf(`(lambda (priv pub) (letrec ((cat-and-hash (lambda (a b)
                                    			(eval (cons 'coproc_blake2s (cons a (cons b nil)))))))
											(= (cat-and-hash priv pub) 0x%x)))`, h)

		proof, err := zk.Prove(program, zk.Expr(fmt.Sprintf("0x%x", left)), zk.Expr(fmt.Sprintf("0x%x", right)))
		assert.NoError(t, err)

		valid, err := zk.Verify(program, zk.Expr(fmt.Sprintf("0x%x", right)), proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("checksig", func(t *testing.T) {
		sk, pk, err := crypto.GenerateNovaKey(rand.Reader)
		assert.NoError(t, err)

		message := []byte("some message")
		sigHash := hash.HashFunc(message)

		sig, err := sk.Sign(sigHash)
		assert.NoError(t, err)

		pkX, pkY := pk.(*crypto.NovaPublicKey).ToXY()
		sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)

		priv := zk.Expr(fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS))
		pub := zk.Expr(fmt.Sprintf("(cons (cons 0x%x (cons 0x%x nil)) (cons 0x%x nil))", pkX, pkY, sigHash))

		program := `(lambda (priv pub) (letrec ((checksig (lambda (sig pubkey sighash)
                                    (eval (cons 'coproc_checksig (cons (car sig) (cons (car (cdr sig)) (cons (car (cdr (cdr sig))) (cons (car pubkey) (cons (car (cdr pubkey)) (cons sighash nil)))))))))))
                            (checksig priv (car pub) (car (cdr pub)))))`

		proof, err := zk.Prove(program, priv, pub)
		assert.NoError(t, err)

		valid, err := zk.Verify(program, pub, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})
}

func TestStandardValidation(t *testing.T) {
	sk, pk, err := crypto.GenerateNovaKey(rand.Reader)
	assert.NoError(t, err)

	_, viewPub, err := crypto.GenerateCurve25519Key(rand.Reader)
	assert.NoError(t, err)

	pkx, pky := pk.(*crypto.NovaPublicKey).ToXY()

	outputLS := types.LockingScript{
		ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
		LockingParams:    [][]byte{pkx, pky},
	}

	scriptHash, err := outputLS.Hash()
	assert.NoError(t, err)

	inSalt, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	outSalt, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	outNote := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     1000000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(outSalt[:]),
		State:      nil,
	}

	outCommitment, err := outNote.Commitment()
	assert.NoError(t, err)

	ser, err := outNote.Serialize()
	assert.NoError(t, err)

	inNote := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     1100000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(inSalt[:]),
		State:      nil,
	}

	inCommitment, err := inNote.Commitment()
	assert.NoError(t, err)

	ciphertext, err := viewPub.(*crypto.Curve25519PublicKey).Encrypt(ser)
	assert.NoError(t, err)

	sigHash, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	sig, err := sk.Sign(sigHash[:])
	assert.NoError(t, err)
	sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)

	acc := blockchain.NewAccumulator()
	for i := 0; i < 10000; i++ {
		r, err := zk.RandomFieldElement()
		assert.NoError(t, err)
		acc.Insert(r[:], false)
	}
	acc.Insert(inCommitment.Bytes(), true)

	icProof, err := acc.GetProof(inCommitment.Bytes())
	assert.NoError(t, err)

	priv := &circparams.PrivateParams{
		Inputs: []circparams.PrivateInput{
			{
				ScriptHash:      scriptHash,
				Amount:          1100000,
				AssetID:         types.IlliumCoinID,
				State:           nil,
				Salt:            inSalt,
				CommitmentIndex: icProof.Index,
				InclusionProof: circparams.InclusionProof{
					Hashes: icProof.Hashes,
					Flags:  icProof.Flags,
				},
				LockingFunction: zk.BasicTransferScript(),
				LockingParams:   [][]byte{pkx, pky},
				UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
			},
		},
		Outputs: []circparams.PrivateOutput{
			{
				ScriptHash: scriptHash,
				Amount:     1000000,
				AssetID:    types.IlliumCoinID,
				State:      nil,
				Salt:       types.NewID(outSalt[:]),
			},
		},
	}

	pub := &circparams.PublicParams{
		Nullifiers: nil,
		TXORoot:    types.ID{},
		Fee:        100000,
		Coinbase:   0,
		MintID:     types.ID{},
		MintAmount: 0,
		Outputs: []circparams.PublicOutput{
			{
				Commitment: outCommitment,
				CipherText: ciphertext,
			},
		},
		SigHash:           types.NewID(sigHash[:]),
		Locktime:          time.Now(),
		LocktimePrecision: 0,
	}

	_, err = zk.Prove(zk.StandardValidationProgram(), priv, pub)
	assert.NoError(t, err)
}
