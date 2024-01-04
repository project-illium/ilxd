// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"crypto/rand"
	"fmt"
	"github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	LoadZKPublicParameters()
	os.Exit(m.Run())
}

func TestProve(t *testing.T) {
	r, err := randomFieldElement()
	assert.NoError(t, err)
	h, err := LurkCommit(fmt.Sprintf("0x%x", r))
	assert.NoError(t, err)

	program := "(lambda (priv pub) (= (num (commit priv)) pub))"

	proof, err := Prove(program, Expr(fmt.Sprintf("0x%x", r)), Expr(fmt.Sprintf("0x%x", h)))
	assert.NoError(t, err)

	valid, err := Verify(program, Expr(fmt.Sprintf("0x%x", h)), proof)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestCoprocessors(t *testing.T) {
	LoadZKPublicParameters()

	t.Run("xor", func(t *testing.T) {
		program := `(lambda (priv pub) (letrec ((xor (lambda (a b)
                                    (eval (cons 'coproc_xor (cons a (cons b nil)))))))
                            (= (xor priv pub) 6)))`

		proof, err := Prove(program, Expr("3"), Expr("5"))
		assert.NoError(t, err)

		valid, err := Verify(program, Expr("5"), proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("blake2s", func(t *testing.T) {
		left, err := randomFieldElement()
		assert.NoError(t, err)
		right, err := randomFieldElement()
		assert.NoError(t, err)

		h := hash.HashMerkleBranches(left[:], right[:])

		program := fmt.Sprintf(`(lambda (priv pub) (letrec ((cat-and-hash (lambda (a b)
                                    (eval (cons 'coproc_blake2s (cons a (cons b nil)))))))
                            (= (cat-and-hash priv pub) 0x%x)))`, h)

		proof, err := Prove(program, Expr(fmt.Sprintf("0x%x", left)), Expr(fmt.Sprintf("0x%x", right)))
		assert.NoError(t, err)

		valid, err := Verify(program, Expr(fmt.Sprintf("0x%x", right)), proof)
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

		priv := Expr(fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS))
		pub := Expr(fmt.Sprintf("(cons (cons 0x%x (cons 0x%x nil)) (cons 0x%x nil))", pkX, pkY, sigHash))

		program := `(lambda (priv pub) (letrec ((checksig (lambda (sig pubkey sighash)
                                    (eval (cons 'coproc_checksig (cons (car sig) (cons (car (cdr sig)) (cons (car (cdr (cdr sig))) (cons (car pubkey) (cons (car (cdr pubkey)) (cons sighash nil)))))))))))
                            (checksig priv (car pub) (car (cdr pub)))))`

		proof, err := Prove(program, priv, pub)
		assert.NoError(t, err)

		valid, err := Verify(program, pub, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	})
}
