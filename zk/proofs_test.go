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

func TestEval(t *testing.T) {
	program := "(lambda (priv pub) (= (+ priv pub) 5))"
	tag, out, _, err := zk.Eval(program, zk.Expr("3"), zk.Expr("2"))
	assert.NoError(t, err)
	assert.Equal(t, zk.TagSym, tag)
	assert.Equal(t, zk.OutputTrue, out)
}

func TestStandardValidation(t *testing.T) {
	tests := []struct {
		Name           string
		Setup          func() (string, zk.Parameters, zk.Parameters, error)
		ExpectedTag    zk.Tag
		ExpectedOutput []byte
	}{
		{
			Name: "standard 1 input, 1 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 1 input, 2 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 2100000}
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 2, opts)
				if err != nil {
					return "", nil, nil, err
				}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 1 input, 3 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 3100000}
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 3, opts)
				if err != nil {
					return "", nil, nil, err
				}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 2 input, 2 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(2, 2, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 3 input, 3 output valid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(3, 3, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard: out amount exceeds in amount",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				opts.outAmounts = map[int]types.Amount{0: 1000001}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 0
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: out amount exceeds in amount plus fee",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				opts := defaultOpts()
				opts.inAmounts = map[int]types.Amount{0: 1000000}
				opts.outAmounts = map[int]types.Amount{0: 1000000}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 1
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: invalid output commitment",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				priv.Outputs[0].State = types.State{[]byte{0x01}}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: invalid input commitment",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				priv.Inputs[0].State = types.State{[]byte{0x01}}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: invalid input index",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				priv.Inputs[0].CommitmentIndex = 1234
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: invalid input proof",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				priv.Inputs[0].InclusionProof.Hashes[0] = r[:]
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: invalid nullifier",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				pub.Nullifiers[0] = types.NewNullifier(r[:])
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: locking script invalid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				priv.Inputs[0].LockingFunction = "(lambda (a b c d e) t)"
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: locking params invalid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				priv.Inputs[0].LockingParams = [][]byte{r[:]}
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard: unlocking params invalid",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				priv, pub, err := generateTxParams(1, 1, defaultOpts())
				if err != nil {
					return "", nil, nil, err
				}
				sk, _, err := crypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return "", nil, nil, err
				}
				sig, err := sk.Sign([]byte("hello"))
				if err != nil {
					return "", nil, nil, err
				}
				sigRx, sigRy, sigS := crypto.UnmarshalSignature(sig)
				priv.Inputs[0].UnlockingParams = fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS)
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard 2 input, 2 output with asset IDs",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				id := types.NewID(r[:])
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: id}
				opts.outAssets = map[int]types.ID{0: id, 1: id}
				priv, pub, err := generateTxParams(2, 2, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 0
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 3 input, 3 output with mixed asset IDs",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				r2, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				id := types.ID{}
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				opts.outAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				priv, pub, err := generateTxParams(3, 3, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 0
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagSym,
			ExpectedOutput: zk.OutputTrue,
		},
		{
			Name: "standard 1 input, 1 output invalid out amount",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: types.NewID(r[:])}
				priv, pub, err := generateTxParams(1, 1, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 0
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard 3 input, 3 output asset output exceeds input",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				r2, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				id := types.ID{}
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				opts.outAssets = map[int]types.ID{0: id, 1: types.NewID(r[:]), 2: types.NewID(r2[:])}
				opts.inAmounts = map[int]types.Amount{1: 1000000}
				opts.outAmounts = map[int]types.Amount{1: 1000001}
				priv, pub, err := generateTxParams(3, 3, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 0
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
		{
			Name: "standard 2 input, 2 output with asset ID and fee without ilx",
			Setup: func() (string, zk.Parameters, zk.Parameters, error) {
				r, err := zk.RandomFieldElement()
				if err != nil {
					return "", nil, nil, err
				}
				id := types.NewID(r[:])
				opts := defaultOpts()
				opts.inAssets = map[int]types.ID{0: id, 1: id}
				opts.outAssets = map[int]types.ID{0: id, 1: id}
				priv, pub, err := generateTxParams(2, 2, opts)
				if err != nil {
					return "", nil, nil, err
				}
				pub.Fee = 1000
				return zk.StandardValidationProgram(), priv, pub, nil
			},
			ExpectedTag:    zk.TagNil,
			ExpectedOutput: zk.OutputFalse,
		},
	}

	for _, test := range tests {
		program, priv, pub, err := test.Setup()
		assert.NoError(t, err)

		tag, val, _, err := zk.Eval(program, priv, pub)
		assert.NoErrorf(t, err, "Test: %s: error: %s", test.Name, err)
		assert.Equalf(t, test.ExpectedTag, tag, "Test %s: Expected tag: %d, got %d", test.Name, test.ExpectedTag, tag)
		assert.Equal(t, test.ExpectedOutput, val, "Test %s: Expected output: %x, got %x", test.Name, test.ExpectedOutput, val)
	}
}

type options struct {
	inAssets   map[int]types.ID
	outAssets  map[int]types.ID
	inAmounts  map[int]types.Amount
	outAmounts map[int]types.Amount
}

func defaultOpts() *options {
	return &options{
		inAssets:   make(map[int]types.ID),
		outAssets:  make(map[int]types.ID),
		inAmounts:  make(map[int]types.Amount),
		outAmounts: make(map[int]types.Amount),
	}
}

func generateTxParams(numInputs, numOutputs int, opts *options) (*circparams.PrivateParams, *circparams.PublicParams, error) {
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

	priv := &circparams.PrivateParams{}
	pub := &circparams.PublicParams{
		SigHash:           sigHash,
		Nullifiers:        nil,
		TXORoot:           types.ID{},
		Fee:               100000,
		Coinbase:          0,
		MintID:            types.ID{},
		MintAmount:        0,
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
			ScriptHash:      note.ScriptHash,
			Amount:          note.Amount,
			AssetID:         note.AssetID,
			Salt:            note.Salt,
			State:           note.State,
			CommitmentIndex: proof.Index,
			LockingFunction: zk.BasicTransferScript(),
			LockingParams:   [][]byte{pkx, pky},
			UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
		})

		nullifier, err := types.CalculateNullifier(proof.Index, note.Salt, zk.BasicTransferScriptCommitment(), [][]byte{pkx, pky}...)
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
