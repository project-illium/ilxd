// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk_test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	lcrypto "github.com/libp2p/go-libp2p/core/crypto"
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

func BenchmarkOneInputTwoOutput(b *testing.B) {
	opts := defaultOpts()
	opts.inAmounts = map[int]types.Amount{0: 2100000}
	opts.outAmounts = map[int]types.Amount{0: 1000000}
	priv, pub, err := generateTxParams(1, 2, opts)
	assert.NoError(b, err)

	_, _, iterations, err := zk.Eval(zk.StandardValidationProgram(), priv, pub)
	assert.NoError(b, err)

	start := time.Now()
	_, err = zk.Prove(zk.StandardValidationProgram(), priv, pub)
	end := time.Since(start)
	assert.NoError(b, err)
	fmt.Printf("Iterations: %d\n", iterations)
	fmt.Printf("Iterations per second: %f\n", (float64(iterations) / float64(end.Milliseconds()) * 1000))
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
		fmt.Println(hex.EncodeToString(r[:]))

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

func TestEval(t *testing.T) {
	program := "(lambda (priv pub) (= (+ priv pub) 5))"
	tag, out, _, err := zk.Eval(program, zk.Expr("3"), zk.Expr("2"))
	assert.NoError(t, err)
	assert.Equal(t, zk.TagSym, tag)
	assert.Equal(t, zk.OutputTrue, out)
}

func TestTransactionProofValidation(t *testing.T) {
	tests := []struct {
		Name           string
		Setup          func() ([]string, zk.Parameters, zk.Parameters, error)
		ExpectedTag    zk.Tag
		ExpectedOutput []byte
	}{
		{
			Name: "standard/mint 1 input, 1 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint 1 input, 2 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 2100000}
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint 1 input, 3 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 3100000}
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 3, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint 2 input, 2 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(2, 2, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint 3 input, 3 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(3, 3, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint: out amount exceeds in amount",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				opts.outAmounts = map[int]types.Amount{0: 1000001}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 0
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: out amount exceeds in amount plus fee",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 1
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: invalid output commitment",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				priv.Outputs[0].State = types.State{[]byte{0x01}}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: invalid input commitment",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				priv.Inputs[0].State = types.State{[]byte{0x01}}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: invalid input index",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				priv.Inputs[0].CommitmentIndex = 1234
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: invalid input proof",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				priv.Inputs[0].InclusionProof.Hashes[0] = r[:]
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: invalid nullifier",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Nullifiers[0] = types.NewNullifier(r[:])
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: locking script invalid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				priv.Inputs[0].Script = "(lambda (a b c d e) t)"
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: locking params invalid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				priv.Inputs[0].LockingParams = [][]byte{r[:]}
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint: unlocking params invalid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return nil, nil, nil, err
				}
				sk, _, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				sig, err := sk.Sign([]byte("hello"))
				if err != nil {
					return nil, nil, nil, err
				}
				sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)
				priv.Inputs[0].UnlockingParams = fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS)
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint 2 input, 2 output with asset IDs",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				id := types.NewID(r[:])
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: id}
				opts.outAssets = map[int]types.ID{0: id, 1: id}
				priv, pub, err := generateTxParams(2, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 0
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint 3 input, 3 output with mixed asset IDs",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				r2, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				id := types.ID{}
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				opts.outAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				priv, pub, err := generateTxParams(3, 3, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 0
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard/mint 1 input, 1 output invalid out amount",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: types.NewID(r[:])}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 0
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint 3 input, 3 output asset output exceeds input",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				r2, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				id := types.ID{}
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				opts.outAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				opts.inAmounts = map[int]types.Amount{1: 1000000}
				opts.outAmounts = map[int]types.Amount{1: 1000001}
				priv, pub, err := generateTxParams(3, 3, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 0
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard/mint 2 input, 2 output with asset ID and fee without ilx",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				id := types.NewID(r[:])
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: id}
				opts.outAssets = map[int]types.ID{0: id, 1: id}
				priv, pub, err := generateTxParams(2, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Fee = 1000
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "mint 1 input, 1 output mint asset with fee valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				opts.outAssets = map[int]types.ID{0: types.NewID(r[:])}
				opts.inAmounts = map[int]types.Amount{0: 50000}
				opts.inAssets = map[int]types.ID{0: types.ID{}}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				mintPub := &circparams.MintPublicParams{
					SigHash:           pub.SigHash,
					Nullifiers:        pub.Nullifiers,
					TXORoot:           pub.TXORoot,
					Fee:               50000,
					MintID:            types.NewID(r[:]),
					MintAmount:        1000000,
					Outputs:           pub.Outputs,
					Locktime:          pub.Locktime,
					LocktimePrecision: pub.LocktimePrecision,
				}
				return []string{zk.MintValidationProgram()}, priv, mintPub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "mint 1 input, 2 output mint asset with fee valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000, 1: 50000}
				opts.outAssets = map[int]types.ID{0: types.NewID(r[:]), 1: types.ID{}}
				opts.inAmounts = map[int]types.Amount{0: 100000}
				opts.inAssets = map[int]types.ID{0: types.ID{}}
				priv, pub, err := generateTxParams(1, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				mintPub := &circparams.MintPublicParams{
					SigHash:           pub.SigHash,
					Nullifiers:        pub.Nullifiers,
					TXORoot:           pub.TXORoot,
					Fee:               50000,
					MintID:            types.NewID(r[:]),
					MintAmount:        1000000,
					Outputs:           pub.Outputs,
					Locktime:          pub.Locktime,
					LocktimePrecision: pub.LocktimePrecision,
				}
				return []string{zk.MintValidationProgram()}, priv, mintPub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "mint 0 input, 1 output mint asset no fee valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				opts.outAssets = map[int]types.ID{0: types.NewID(r[:])}
				priv, pub, err := generateTxParams(0, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				mintPub := &circparams.MintPublicParams{
					SigHash:           pub.SigHash,
					Nullifiers:        pub.Nullifiers,
					TXORoot:           pub.TXORoot,
					Fee:               0,
					MintID:            types.NewID(r[:]),
					MintAmount:        1000000,
					Outputs:           pub.Outputs,
					Locktime:          pub.Locktime,
					LocktimePrecision: pub.LocktimePrecision,
				}
				return []string{zk.MintValidationProgram()}, priv, mintPub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "mint 0 input, 1 output mint asset no fee too many coins",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				opts.outAssets = map[int]types.ID{0: types.NewID(r[:])}
				priv, pub, err := generateTxParams(0, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				mintPub := &circparams.MintPublicParams{
					SigHash:           pub.SigHash,
					Nullifiers:        pub.Nullifiers,
					TXORoot:           pub.TXORoot,
					Fee:               0,
					MintID:            types.NewID(r[:]),
					MintAmount:        100000,
					Outputs:           pub.Outputs,
					Locktime:          pub.Locktime,
					LocktimePrecision: pub.LocktimePrecision,
				}
				return []string{zk.MintValidationProgram()}, priv, mintPub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "mint 1 input, 2 output mint asset with fee too many coins",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000, 1: 50000}
				opts.outAssets = map[int]types.ID{0: types.NewID(r[:]), 1: types.ID{}}
				opts.inAmounts = map[int]types.Amount{0: 100000}
				opts.inAssets = map[int]types.ID{0: types.ID{}}
				priv, pub, err := generateTxParams(1, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				mintPub := &circparams.MintPublicParams{
					SigHash:           pub.SigHash,
					Nullifiers:        pub.Nullifiers,
					TXORoot:           pub.TXORoot,
					Fee:               50000,
					MintID:            types.NewID(r[:]),
					MintAmount:        100000,
					Outputs:           pub.Outputs,
					Locktime:          pub.Locktime,
					LocktimePrecision: pub.LocktimePrecision,
				}
				return []string{zk.MintValidationProgram()}, priv, mintPub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "coinbase 1 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(0, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				cbPriv := circparams.CoinbasePrivateParams(priv.Outputs)
				cbPub := &circparams.CoinbasePublicParams{
					Coinbase: 1000000,
					Outputs:  pub.Outputs,
				}
				return []string{zk.CoinbaseValidationProgram()}, &cbPriv, cbPub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "coinbase 2 output valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(0, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				cbPriv := circparams.CoinbasePrivateParams(priv.Outputs)
				cbPub := &circparams.CoinbasePublicParams{
					Coinbase: 2000000,
					Outputs:  pub.Outputs,
				}
				return []string{zk.CoinbaseValidationProgram()}, &cbPriv, cbPub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "coinbase 2 output too many coins",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(0, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				cbPriv := circparams.CoinbasePrivateParams(priv.Outputs)
				cbPub := &circparams.CoinbasePublicParams{
					Coinbase: 1999999,
					Outputs:  pub.Outputs,
				}
				return []string{zk.CoinbaseValidationProgram()}, &cbPriv, cbPub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "coinbase 2 output non-ilx asset",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				opts.outAssets = map[int]types.ID{0: types.ID{}, 1: types.NewID(r[:])}
				priv, pub, err := generateTxParams(0, 2, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				cbPriv := circparams.CoinbasePrivateParams(priv.Outputs)
				cbPub := &circparams.CoinbasePublicParams{
					Coinbase: 1000000,
					Outputs:  pub.Outputs,
				}
				return []string{zk.CoinbaseValidationProgram()}, &cbPriv, cbPub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "coinbase 2 output invalid output commitment",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(0, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				cbPriv := circparams.CoinbasePrivateParams(priv.Outputs)
				cbPub := &circparams.CoinbasePublicParams{
					Coinbase: 1000000,
					Outputs:  pub.Outputs,
				}
				cbPriv[0].State = types.State{[]byte{0x01}}
				return []string{zk.CoinbaseValidationProgram()}, &cbPriv, cbPub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "stake invalid asset ID",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				opts.inAssets = map[int]types.ID{0: types.NewID(r[:])}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake private doesn't equal public amount",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1100000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake invalid input commitment index",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				privIn.CommitmentIndex = 0
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake invalid input commitment hashes",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				privIn.InclusionProof.Hashes[0] = r[:]
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake invalid nullifier",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				r, err := zk.RandomFieldElement()
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub.Nullifier = types.NewNullifier(r[:])
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake invalid commmitment",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				privIn.State = types.State{[]byte{0x01}}
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake invalid unlocking script",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				sk, _, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				sig, err := sk.Sign([]byte("hello"))
				if err != nil {
					return nil, nil, nil, err
				}
				sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)
				privIn.UnlockingParams = fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS)
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake invalid locking function",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				privIn.Script = "(lambda (a b c d e) t)"
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "stake timelocked with wrong script",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 0, opts)
				if err != nil {
					return nil, nil, nil, err
				}
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				stakePub.LockedUntil = time.Now()
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard 1 of 1 multisig input valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				commitment, err := zk.LurkCommit(zk.MultisigScript())
				if err != nil {
					return nil, nil, nil, err
				}
				sk, pk, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				pkX, pkY := pk.(*crypto.NovaPublicKey).ToXY()
				opts := defaultOpts()
				opts.inLockingParams = map[int][][]byte{0: {{0x01}, pkX, pkY}}
				opts.inScriptCommitments = map[int]types.ID{0: types.NewID(commitment)}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}

				sig, err := sk.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				unlockingScript, err := zk.MakeMultisigUnlockingParams([]lcrypto.PubKey{pk}, [][]byte{sig}, pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}

				priv.Inputs[0].UnlockingParams = unlockingScript
				priv.Inputs[0].Script = zk.MultisigScript()

				nullifer, err := types.CalculateNullifier(priv.Inputs[0].CommitmentIndex, priv.Inputs[0].Salt, commitment, priv.Inputs[0].LockingParams...)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Nullifiers[0] = nullifer
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 2 of 3 multisig input valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				commitment, err := zk.LurkCommit(zk.MultisigScript())
				if err != nil {
					return nil, nil, nil, err
				}
				sk1, pk1, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				sk2, pk2, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				_, pk3, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				pk1X, pk1Y := pk1.(*crypto.NovaPublicKey).ToXY()
				pk2X, pk2Y := pk2.(*crypto.NovaPublicKey).ToXY()
				pk3X, pk3Y := pk3.(*crypto.NovaPublicKey).ToXY()
				opts := defaultOpts()
				opts.inLockingParams = map[int][][]byte{0: {{0x02}, pk1X, pk1Y, pk2X, pk2Y, pk3X, pk3Y}}
				opts.inScriptCommitments = map[int]types.ID{0: types.NewID(commitment)}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}

				sig1, err := sk1.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				sig2, err := sk2.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				unlockingScript, err := zk.MakeMultisigUnlockingParams([]lcrypto.PubKey{pk1, pk2, pk3}, [][]byte{sig1, sig2}, pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}

				priv.Inputs[0].UnlockingParams = unlockingScript
				priv.Inputs[0].Script = zk.MultisigScript()

				nullifer, err := types.CalculateNullifier(priv.Inputs[0].CommitmentIndex, priv.Inputs[0].Salt, commitment, priv.Inputs[0].LockingParams...)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Nullifiers[0] = nullifer
				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "stake 2 of 3 multisig timelock valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				sk1, pk1, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				sk2, pk2, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				_, pk3, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}

				pk1X, pk1Y := pk1.(*crypto.NovaPublicKey).ToXY()
				pk2X, pk2Y := pk2.(*crypto.NovaPublicKey).ToXY()
				pk3X, pk3Y := pk3.(*crypto.NovaPublicKey).ToXY()
				opts := defaultOpts()
				locktime := time.Now()
				locktimeBytes := make([]byte, 8)
				binary.BigEndian.PutUint64(locktimeBytes, uint64(locktime.Unix()))
				opts.inLockingParams = map[int][][]byte{0: {locktimeBytes, {0x02}, pk1X, pk1Y, pk2X, pk2Y, pk3X, pk3Y}}
				opts.inScriptCommitments = map[int]types.ID{0: types.NewID(zk.TimelockedMultisigScriptCommitment())}
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}

				sig1, err := sk1.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				sig2, err := sk2.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				unlockingScript, err := zk.MakeMultisigUnlockingParams([]lcrypto.PubKey{pk1, pk2, pk3}, [][]byte{sig1, sig2}, pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}

				priv.Inputs[0].UnlockingParams = unlockingScript
				priv.Inputs[0].Script = zk.TimelockedMultisigScript()

				nullifer, err := types.CalculateNullifier(priv.Inputs[0].CommitmentIndex, priv.Inputs[0].Salt, zk.TimelockedMultisigScriptCommitment(), priv.Inputs[0].LockingParams...)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Nullifiers[0] = nullifer
				pub.Locktime = locktime
				pub.LocktimePrecision = 600 * time.Second
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "stake 2 of 3 multisig timelock invalid locktime",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				sk1, pk1, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				sk2, pk2, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				_, pk3, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				pk1X, pk1Y := pk1.(*crypto.NovaPublicKey).ToXY()
				pk2X, pk2Y := pk2.(*crypto.NovaPublicKey).ToXY()
				pk3X, pk3Y := pk3.(*crypto.NovaPublicKey).ToXY()
				opts := defaultOpts()
				locktime := time.Now()
				locktimeBytes := make([]byte, 8)
				binary.BigEndian.PutUint64(locktimeBytes, uint64(locktime.Unix()))
				opts.inLockingParams = map[int][][]byte{0: {locktimeBytes, {0x02}, pk1X, pk1Y, pk2X, pk2Y, pk3X, pk3Y}}
				opts.inScriptCommitments = map[int]types.ID{0: types.NewID(zk.TimelockedMultisigScriptCommitment())}
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}

				sig1, err := sk1.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				sig2, err := sk2.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}
				unlockingScript, err := zk.MakeMultisigUnlockingParams([]lcrypto.PubKey{pk1, pk2, pk3}, [][]byte{sig1, sig2}, pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}

				priv.Inputs[0].UnlockingParams = unlockingScript
				priv.Inputs[0].Script = zk.TimelockedMultisigScript()

				nullifer, err := types.CalculateNullifier(priv.Inputs[0].CommitmentIndex, priv.Inputs[0].Salt, zk.TimelockedMultisigScriptCommitment(), priv.Inputs[0].LockingParams...)
				if err != nil {
					return nil, nil, nil, err
				}
				pub.Nullifiers[0] = nullifer
				pub.Locktime = locktime.Add(-time.Minute)
				pub.LocktimePrecision = 600 * time.Second
				stakePub := &circparams.StakePublicParams{
					StakeAmount: 1000000,
					SigHash:     pub.SigHash,
					Nullifier:   pub.Nullifiers[0],
					TXORoot:     pub.TXORoot,
					LockedUntil: pub.Locktime,
				}
				privIn := circparams.StakePrivateParams(priv.Inputs[0])
				return []string{zk.StakeValidationProgram()}, &privIn, stakePub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "public address valid",
			Setup: func() ([]string, zk.Parameters, zk.Parameters, error) {
				sk1, pk1, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				_, pk2, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, nil, err
				}
				pk1X, pk1Y := pk1.(*crypto.NovaPublicKey).ToXY()
				pk2X, pk2Y := pk2.(*crypto.NovaPublicKey).ToXY()

				lockingParams := fmt.Sprintf("(cons 1 (cons 0x%x (cons 0x%x nil)))", pk1X, pk1Y)
				lpHash, err := zk.LurkCommit(lockingParams)
				if err != nil {
					return nil, nil, nil, err
				}

				lockingParams2 := fmt.Sprintf("(cons 1 (cons 0x%x (cons 0x%x nil)))", pk2X, pk2Y)
				lpHash2, err := zk.LurkCommit(lockingParams2)
				if err != nil {
					return nil, nil, nil, err
				}

				lockingScript := types.LockingScript{
					ScriptCommitment: types.NewID(zk.PublicAddressScriptCommitment()),
					LockingParams:    nil,
				}

				scriptHash, err := lockingScript.Hash()
				if err != nil {
					return nil, nil, nil, err
				}

				opts := defaultOpts()
				opts.inScriptCommitments = map[int]types.ID{0: types.NewID(zk.PublicAddressScriptCommitment())}
				opts.inLockingParams = map[int][][]byte{0: nil}
				opts.inState = map[int]types.State{0: types.State{lpHash}}

				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return nil, nil, nil, err
				}

				sig, err := sk1.Sign(pub.SigHash.Bytes())
				if err != nil {
					return nil, nil, nil, err
				}

				sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)

				priv.Inputs[0].Script = zk.PublicAddressScript()
				priv.Inputs[0].LockingParams = nil
				priv.Inputs[0].UnlockingParams = fmt.Sprintf("(cons %s (cons (cons 1 nil) (cons (cons 0x%x (cons 0x%x (cons 0x%x nil))) nil)))", lockingParams, sigRx, sigRy, sigS)

				priv.Outputs[0].ScriptHash = scriptHash
				priv.Outputs[0].State = types.State{lpHash2}

				outNote := types.SpendNote{
					ScriptHash: scriptHash,
					Amount:     priv.Outputs[0].Amount,
					AssetID:    priv.Outputs[0].AssetID,
					Salt:       priv.Outputs[0].Salt,
					State:      priv.Outputs[0].State,
				}

				outCommitment, err := outNote.Commitment()
				if err != nil {
					return nil, nil, nil, err
				}

				ser, err := outNote.ToPublicCiphertext()
				if err != nil {
					return nil, nil, nil, err
				}

				pub.Outputs[0].Commitment = outCommitment
				pub.Outputs[0].CipherText = ser

				return []string{zk.StandardValidationProgram(), zk.MintValidationProgram()}, priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
	}

	for _, test := range tests {
		programs, priv, pub, err := test.Setup()
		assert.NoError(t, err)

		for _, prog := range programs {
			tag, val, _, err := zk.Eval(prog, priv, pub)
			assert.NoErrorf(t, err, "Test: %s: error: %s", test.Name, err)
			assert.Equalf(t, test.ExpectedTag, tag, "Test %s: Expected tag: %d, got %d", test.Name, test.ExpectedTag, tag)
			assert.Equal(t, test.ExpectedOutput, val, "Test %s: Expected output: %x, got %x", test.Name, test.ExpectedOutput, val)
		}
	}
}

type options struct {
	inAssets            map[int]types.ID
	outAssets           map[int]types.ID
	inAmounts           map[int]types.Amount
	outAmounts          map[int]types.Amount
	inState             map[int]types.State
	inScriptCommitments map[int]types.ID
	inLockingParams     map[int][][]byte
}

func defaultOpts() *options {
	return &options{
		inAssets:            make(map[int]types.ID),
		outAssets:           make(map[int]types.ID),
		inAmounts:           make(map[int]types.Amount),
		outAmounts:          make(map[int]types.Amount),
		inState:             make(map[int]types.State),
		inScriptCommitments: make(map[int]types.ID),
		inLockingParams:     make(map[int][][]byte),
	}
}

func generateTxParams(numInputs, numOutputs int, opts *options) (*circparams.StandardPrivateParams, *circparams.StandardPublicParams, error) {
	sigHash, err := zk.RandomFieldElement()
	if err != nil {
		return nil, nil, err
	}

	acc := blockchain.NewAccumulator()
	for i := 0; i < 10000; i++ {
		r, err := zk.RandomFieldElement()
		if err != nil {
			return nil, nil, err
		}
		acc.Insert(r[:], false)
	}

	priv := &circparams.StandardPrivateParams{}
	pub := &circparams.StandardPublicParams{
		SigHash:           sigHash,
		Nullifiers:        nil,
		TXORoot:           types.ID{},
		Fee:               100000,
		Outputs:           nil,
		Locktime:          time.Time{},
		LocktimePrecision: 0,
	}

	var inputCommitments []types.ID

	for i := 0; i < numOutputs; i++ {
		_, viewPub, err := crypto.GenerateCurve25519Key(rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		_, pk, err := crypto.GenerateNovaKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		pkx, pky := pk.(*crypto.NovaPublicKey).ToXY()

		lockingScript := types.LockingScript{
			ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
			LockingParams:    [][]byte{pkx, pky},
		}

		scriptHash, err := lockingScript.Hash()
		if err != nil {
			return nil, nil, err
		}

		salt, err := zk.RandomFieldElement()
		if err != nil {
			return nil, nil, err
		}

		note := types.SpendNote{
			ScriptHash: scriptHash,
			Amount:     1000000,
			AssetID:    types.IlliumCoinID,
			Salt:       types.NewID(salt[:]),
			State:      nil,
		}

		amt, ok := opts.outAmounts[i]
		if ok {
			note.Amount = amt
		}
		assetID, ok := opts.outAssets[i]
		if ok {
			note.AssetID = assetID
		}

		serializedNote, err := note.Serialize()
		if err != nil {
			return nil, nil, err
		}

		commitment, err := note.Commitment()
		if err != nil {
			return nil, nil, err
		}
		priv.Outputs = append(priv.Outputs, circparams.PrivateOutput{
			ScriptHash: note.ScriptHash,
			Amount:     note.Amount,
			AssetID:    note.AssetID,
			Salt:       note.Salt,
			State:      note.State,
		})

		ciphtertext, err := viewPub.(*crypto.Curve25519PublicKey).Encrypt(serializedNote)
		if err != nil {
			return nil, nil, err
		}

		pub.Outputs = append(pub.Outputs, circparams.PublicOutput{
			Commitment: commitment,
			CipherText: ciphtertext,
		})
	}

	for i := 0; i < numInputs; i++ {
		sk, pk, err := crypto.GenerateNovaKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		pkx, pky := pk.(*crypto.NovaPublicKey).ToXY()

		lockingScript := types.LockingScript{
			ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
			LockingParams:    [][]byte{pkx, pky},
		}

		if customCommit, ok := opts.inScriptCommitments[i]; ok {
			lockingScript.ScriptCommitment = customCommit
		}
		if lockingParams, ok := opts.inLockingParams[i]; ok {
			lockingScript.LockingParams = lockingParams
		}

		scriptHash, err := lockingScript.Hash()
		if err != nil {
			return nil, nil, err
		}

		salt, err := zk.RandomFieldElement()
		if err != nil {
			return nil, nil, err
		}

		note := types.SpendNote{
			ScriptHash: scriptHash,
			Amount:     1100000,
			AssetID:    types.IlliumCoinID,
			Salt:       types.NewID(salt[:]),
			State:      nil,
		}

		amt, ok := opts.inAmounts[i]
		if ok {
			note.Amount = amt
		}
		assetID, ok := opts.inAssets[i]
		if ok {
			note.AssetID = assetID
		}
		state, ok := opts.inState[i]
		if ok {
			note.State = state
		}

		commitment, err := note.Commitment()
		if err != nil {
			return nil, nil, err
		}

		inputCommitments = append(inputCommitments, commitment)

		acc.Insert(commitment.Bytes(), true)
		proof, err := acc.GetProof(commitment.Bytes())
		if err != nil {
			return nil, nil, err
		}

		sig, err := sk.Sign(sigHash[:])
		if err != nil {
			return nil, nil, err
		}
		sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)

		priv.Inputs = append(priv.Inputs, circparams.PrivateInput{
			Amount:          note.Amount,
			AssetID:         note.AssetID,
			Salt:            note.Salt,
			State:           note.State,
			CommitmentIndex: proof.Index,
			Script:          zk.BasicTransferScript(),
			LockingParams:   lockingScript.LockingParams,
			UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
		})

		nullifier, err := types.CalculateNullifier(proof.Index, note.Salt, lockingScript.ScriptCommitment.Bytes(), lockingScript.LockingParams...)
		if err != nil {
			return nil, nil, err
		}
		pub.Nullifiers = append(pub.Nullifiers, nullifier)
	}

	pub.TXORoot = acc.Root()
	for i := range priv.Inputs {
		proof, err := acc.GetProof(inputCommitments[i].Bytes())
		if err != nil {
			return nil, nil, err
		}
		priv.Inputs[i].InclusionProof = circparams.InclusionProof{
			Hashes: proof.Hashes,
			Flags:  proof.Flags,
		}
	}
	return priv, pub, nil
}
