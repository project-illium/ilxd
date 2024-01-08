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

	outSalt1, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	outNote1 := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     500000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(outSalt1[:]),
		State:      nil,
	}

	outCommitment1, err := outNote1.Commitment()
	assert.NoError(t, err)

	ser1, err := outNote1.Serialize()
	assert.NoError(t, err)

	ciphertext1, err := viewPub.(*crypto.Curve25519PublicKey).Encrypt(ser1)
	assert.NoError(t, err)

	outSalt2, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	outNote2 := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     500000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(outSalt2[:]),
		State:      nil,
	}

	outCommitment2, err := outNote2.Commitment()
	assert.NoError(t, err)

	ser2, err := outNote2.Serialize()
	assert.NoError(t, err)

	ciphertext2, err := viewPub.(*crypto.Curve25519PublicKey).Encrypt(ser2)
	assert.NoError(t, err)

	outSalt3, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	outNote3 := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     50000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(outSalt3[:]),
		State:      nil,
	}

	outCommitment3, err := outNote3.Commitment()
	assert.NoError(t, err)

	ser3, err := outNote3.Serialize()
	assert.NoError(t, err)

	ciphertext3, err := viewPub.(*crypto.Curve25519PublicKey).Encrypt(ser3)
	assert.NoError(t, err)

	inSalt1, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	inNote1 := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     1100000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(inSalt1[:]),
		State:      nil,
	}

	inCommitment1, err := inNote1.Commitment()
	assert.NoError(t, err)

	inSalt2, err := zk.RandomFieldElement()
	assert.NoError(t, err)

	inNote2 := types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     1100000,
		AssetID:    types.IlliumCoinID,
		Salt:       types.NewID(inSalt2[:]),
		State:      nil,
	}

	inCommitment2, err := inNote2.Commitment()
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
	acc.Insert(inCommitment1.Bytes(), true)
	acc.Insert(inCommitment2.Bytes(), true)
	for i := 0; i < 10000; i++ {
		r, err := zk.RandomFieldElement()
		assert.NoError(t, err)
		acc.Insert(r[:], false)
	}

	icProof1, err := acc.GetProof(inCommitment1.Bytes())
	assert.NoError(t, err)

	inNullifier1, err := types.CalculateNullifier(icProof1.Index, inNote1.Salt, zk.BasicTransferScriptCommitment(), [][]byte{pkx, pky}...)
	assert.NoError(t, err)

	icProof2, err := acc.GetProof(inCommitment2.Bytes())
	assert.NoError(t, err)

	inNullifier2, err := types.CalculateNullifier(icProof2.Index, inNote2.Salt, zk.BasicTransferScriptCommitment(), [][]byte{pkx, pky}...)
	assert.NoError(t, err)

	priv := &circparams.PrivateParams{
		Inputs: []circparams.PrivateInput{
			{
				ScriptHash:      inNote1.ScriptHash,
				Amount:          inNote1.Amount,
				AssetID:         inNote1.AssetID,
				Salt:            inNote1.Salt,
				State:           inNote1.State,
				CommitmentIndex: icProof1.Index,
				InclusionProof: circparams.InclusionProof{
					Hashes: icProof1.Hashes,
					Flags:  icProof1.Flags,
				},
				LockingFunction: zk.BasicTransferScript(),
				LockingParams:   [][]byte{pkx, pky},
				UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
			},
		},
		Outputs: []circparams.PrivateOutput{
			{
				ScriptHash: outNote1.ScriptHash,
				Amount:     outNote1.Amount,
				AssetID:    outNote1.AssetID,
				Salt:       outNote1.Salt,
				State:      outNote1.State,
			},
			{
				ScriptHash: outNote2.ScriptHash,
				Amount:     outNote2.Amount,
				AssetID:    outNote2.AssetID,
				Salt:       outNote2.Salt,
				State:      outNote2.State,
			},
		},
	}

	pub := &circparams.PublicParams{
		SigHash:    types.NewID(sigHash[:]),
		Nullifiers: []types.Nullifier{inNullifier1},
		TXORoot:    acc.Root(),
		Fee:        100000,
		Coinbase:   0,
		MintID:     types.ID{},
		MintAmount: 0,
		Outputs: []circparams.PublicOutput{
			{
				Commitment: outCommitment1,
				CipherText: ciphertext1,
			},
			{
				Commitment: outCommitment2,
				CipherText: ciphertext2,
			},
		},
		Locktime:          time.Now(),
		LocktimePrecision: 0,
	}

	tests := []struct {
		Name           string
		Setup          func() (string, zk.Parameters, zk.Parameters, error)
		ExpectedTag    zk.Tag
		ExpectedOutput []byte
	}{
		{
			Name: "standard 1 input, 2 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 1 input, 1 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv2 := priv.Clone()
				priv2.Outputs = []circparams.PrivateOutput{priv.Outputs[0]}
				pub2 := pub.Clone()
				pub2.Outputs = []circparams.PublicOutput{pub.Outputs[0]}
				return zk.StandardValidationProgram(), priv2, pub2, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 1 input, 3 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv2 := priv.Clone()
				priv2.Outputs = append(priv2.Outputs, circparams.PrivateOutput{
					ScriptHash: outNote3.ScriptHash,
					Amount:     outNote3.Amount,
					AssetID:    outNote3.AssetID,
					Salt:       outNote3.Salt,
					State:      outNote3.State,
				})
				pub2 := pub.Clone()
				pub2.Outputs = append(pub2.Outputs, circparams.PublicOutput{
					Commitment: outCommitment3,
					CipherText: ciphertext3,
				})
				pub2.Fee = 50000
				return zk.StandardValidationProgram(), priv2, pub2, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 2 input, 2 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv2 := priv.Clone()
				priv2.Inputs = append(priv2.Inputs, circparams.PrivateInput{
					ScriptHash:      inNote2.ScriptHash,
					Amount:          inNote2.Amount,
					AssetID:         inNote2.AssetID,
					Salt:            inNote2.Salt,
					State:           inNote2.State,
					CommitmentIndex: icProof2.Index,
					InclusionProof: circparams.InclusionProof{
						Hashes: icProof2.Hashes,
						Flags:  icProof2.Flags,
					},
					LockingFunction: zk.BasicTransferScript(),
					LockingParams:   [][]byte{pkx, pky},
					UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
				})
				pub2 := pub.Clone()
				pub2.Nullifiers = append(pub2.Nullifiers, inNullifier2)
				return zk.StandardValidationProgram(), priv2, pub2, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
	}

	for _, test := range tests {
		program, priv, pub, err := test.Setup()
		assert.NoError(t, err)

		tag, val, _, err := zk.Eval(program, priv, pub)
		assert.NoErrorf(t, err, "Test: %s: error: %s", test.Name, err)
		assert.Equalf(t, test.ExpectedTag, tag, "Test %s: Expected tag: %d, got %d", test.Name, test.ExpectedTag, tag)
		assert.Equal(t, test.ExpectedOutput, val, "Test %s: Expected output: %x, got %x", test.ExpectedOutput, val)
	}

	/*start := time.Now()
	_, err = zk.Prove(zk.StandardValidationProgram(), priv, pub)
	assert.NoError(t, err)
	fmt.Println(time.Since(start))*/

	/*start = time.Now()
	valid, err := zk.Verify(zk.StandardValidationProgram(), pub, proof)
	assert.NoError(t, err)
	assert.True(t, valid)
	fmt.Println(time.Since(start))*/
}

func TestEval(t *testing.T) {
	program := "(lambda (priv pub) (= (+ priv pub) 5))"
	tag, out, _, err := zk.Eval(program, zk.Expr("3"), zk.Expr("2"))
	assert.NoError(t, err)
	assert.Equal(t, zk.TagSym, tag)
	assert.Equal(t, zk.OutputTrue, out)
}
