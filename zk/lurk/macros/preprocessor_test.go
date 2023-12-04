// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros_test

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/project-illium/ilxd/zk/lurk/macros"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

type customStack struct {
	stack *list.List
}

func (c *customStack) Push(value string) {
	c.stack.PushFront(value)
}

func (c *customStack) Pop() error {
	if c.stack.Len() > 0 {
		ele := c.stack.Front()
		c.stack.Remove(ele)
	}
	return fmt.Errorf("Pop Error: Stack is empty")
}

func (c *customStack) Front() (string, error) {
	if c.stack.Len() > 0 {
		if val, ok := c.stack.Front().Value.(string); ok {
			return val, nil
		}
		return "", fmt.Errorf("Peep Error: Stack Datatype is incorrect")
	}
	return "", fmt.Errorf("Peep Error: Stack is empty")
}

func (c *customStack) Size() int {
	return c.stack.Len()
}

func (c *customStack) Empty() bool {
	return c.stack.Len() == 0
}

func isValid(s string) bool {
	customStack := &customStack{
		stack: list.New(),
	}

	for _, val := range s {

		if val == '(' || val == '[' || val == '{' {
			customStack.Push(string(val))
		} else if val == ')' {
			poppedValue, _ := customStack.Front()
			if poppedValue != "(" {
				return false
			}
			customStack.Pop()
		} else if val == ']' {
			poppedValue, _ := customStack.Front()
			if poppedValue != "[" {
				return false
			}
			customStack.Pop()

		} else if val == '}' {
			poppedValue, _ := customStack.Front()
			if poppedValue != "{" {
				return false
			}
			customStack.Pop()

		}

	}

	return customStack.Size() == 0
}

func TestPreProcessValidParentheses(t *testing.T) {
	type testVector struct {
		input    string
		expected string
	}
	tests := []testVector{
		{"(+ x 3)", "(+ x 3)"},
		{"!(def x 3)", "(let ((x 3)))"},
		{"!(def x (car y))", "(let ((x (car y))))"},
		{"!(def x 3) t", "(let ((x 3)) t)"},
		{"!(def x (car y)) t", "(let ((x (car y))) t)"},
		{"!(defrec x 3)", "(letrec ((x 3)))"},
		{"!(defrec x (car y))", "(letrec ((x (car y))))"},
		{"!(defrec x 3) t", "(letrec ((x 3)) t)"},
		{"!(defrec x (car y)) t", "(letrec ((x (car y))) t)"},
		{"!(defun f (x) (+ x 3))", "(letrec ((f (lambda (x) (+ x 3)))))"},
		{"!(defun f (x) (+ x 3)) t", "(letrec ((f (lambda (x) (+ x 3)))) t)"},
		{"!(assert t)", "(if (eq t nil) nil)"},
		{"!(assert (+ x 5)) nil", "(if (eq (+ x 5) nil) nil nil)"},
		{"!(assert t) nil", "(if (eq t nil) nil nil)"},
		{"!(assert-eq x 3)", "(if (eq (eq x 3 ) nil) nil)"},
		{"!(assert-eq x 3) t", "(if (eq (eq x 3 ) nil) nil t)"},
		{"!(defun f (x) (!(assert t) 3))", "(letrec ((f (lambda (x) ((if (eq t nil) nil 3))))))"},
		{"(lambda (script-params unlocking-params input-index private-params public-params) !(assert-eq (+ x 5) 4) !(def z 5) !(assert t) t)", "(lambda (script-params unlocking-params input-index private-params public-params) (if (eq (eq (+ x 5) 4) nil) nil (let ((z 5)) (if (eq t nil) nil t))))"},
		{"!(list 1 2 3 4)", "(cons 1 (cons 2 (cons 3 (cons 4 nil))))"},
		{"!(list 1 (car x) 3 4)", "(cons 1 (cons (car x) (cons 3 (cons 4 nil))))"},
		{"!(param nullifiers 0)", "(nth 0 (nth 0 public-params))"},
		{"!(param txo-root)", "(nth 1 public-params)"},
		{"!(param fee)", "(nth 2 public-params)"},
		{"!(param coinbase)", "(nth 3 public-params)"},
		{"!(param mint-id)", "(nth 4 public-params)"},
		{"!(param mint-amount)", "(nth 5 public-params)"},
		{"!(param sighash)", "(nth 7 public-params)"},
		{"!(param locktime)", "(nth 8 public-params)"},
		{"!(param locktime-precision)", "(nth 9 public-params)"},
		{"!(param priv-in 2)", "(nth 2 (car private-params))"},
		{"!(param priv-out 3)", "(nth 3 (car (cdr private-params)))"},
		{"!(param pub-out 4)", "(nth 4 (nth 6 public-params))"},
		{"!(param priv-in 4 script-hash)", "(hash (cons (car (nth 4 (car private-params))) (cons (nth 3 (nth 4 (car private-params))) nil)))"},
		{"!(param priv-in 2 script-commitment)", "(nth 0 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 amount)", "(nth 1 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 asset-id)", "(nth 2 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 script-params)", "(nth 3 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 commitment-index)", "(nth 4 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 state)", "(nth 5 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 salt)", "(nth 6 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 unlocking-params)", "(nth 7 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 inclusion-proof-hashes)", "(nth 8 (nth 2 (car private-params)))"},
		{"!(param priv-in 2 inclusion-proof-accumulator)", "(nth 9 (nth 2 (car private-params)))"},
		{"!(param priv-out 3 script-hash)", "(nth 0 (nth 3 (car (cdr private-params))))"},
		{"!(param priv-out 3 amount)", "(nth 1 (nth 3 (car (cdr private-params))))"},
		{"!(param priv-out 3 asset-id)", "(nth 2 (nth 3 (car (cdr private-params))))"},
		{"!(param priv-out 3 state)", "(nth 3 (nth 3 (car (cdr private-params))))"},
		{"!(param priv-out 3 salt)", "(nth 4 (nth 3 (car (cdr private-params))))"},
		{"!(param pub-out 4 commitment)", "(nth 0 (nth 4 (nth 6 public-params)))"},
		{"!(param pub-out 4 ciphertext)", "(nth 1 (nth 4 (nth 6 public-params)))"},
	}

	mp, err := macros.NewMacroPreprocessor()
	assert.NoError(t, err)
	for i, test := range tests {
		lurkProgram, err := mp.Preprocess(test.input)
		lurkProgram = strings.ReplaceAll(lurkProgram, "\n", "")
		lurkProgram = strings.ReplaceAll(lurkProgram, "\t", "")
		assert.NoError(t, err)
		assert.Truef(t, isValid(lurkProgram), "Test %d should be valid", i)
		assert.Equalf(t, test.expected, lurkProgram, "Test %d not as expected", i)
	}
}

func TestMacroImports(t *testing.T) {
	tempDir := path.Join(os.TempDir(), "marco_test")
	defer os.Remove(tempDir)

	type module struct {
		path string
		file string
	}
	type testVector struct {
		input    string
		modules  []module
		expected string
	}

	mod1 := `!(module math (
			!(defun plus-two (x) (+ x 2))
			!(defun plus-three (x) (+ x 3))
		))

		!(module time (
			!(assert (<= !(param locktime-precision) 30))
		))
		`

	tests := []testVector{
		{
			input: `!(defun my-func (y) (
				!(import math)
				(plus-two 10)
			))`,
			modules:  []module{{path: filepath.Join(tempDir, "mod.lurk"), file: mod1}},
			expected: "(letrec ((my-func (lambda (y) ((letrec ((plus-two (lambda (x) (+ x 2))))(letrec ((plus-three (lambda (x) (+ x 3))))(plus-two 10))))))))",
		},
		{
			input: `!(defun my-func (y) (
				!(import time)
				(plus-two 10)
			))`,
			modules:  []module{{path: filepath.Join(tempDir, "mod.lurk"), file: mod1}},
			expected: "(letrec ((my-func (lambda (y) ((if (eq (<= (nth 9 public-params) 30) nil) nil(plus-two 10)))))))",
		},
		{
			input: `!(defun my-func (y) (
				!(import std/math)
				(plus-two 10)
			))`,
			modules:  []module{{path: filepath.Join(tempDir, "std", "mod.lurk"), file: mod1}},
			expected: "(letrec ((my-func (lambda (y) ((letrec ((plus-two (lambda (x) (+ x 2))))(letrec ((plus-three (lambda (x) (+ x 3))))(plus-two 10))))))))",
		},
	}

	mp, err := macros.NewMacroPreprocessor(macros.DependencyDir(tempDir))
	assert.NoError(t, err)
	for i, test := range tests {
		for _, mod := range test.modules {
			err := os.MkdirAll(filepath.Dir(mod.path), 0755)
			assert.NoError(t, err)

			err = os.WriteFile(mod.path, []byte(mod.file), 0644)
			assert.NoError(t, err)
		}

		lurkProgram, err := mp.Preprocess(test.input)
		lurkProgram = strings.ReplaceAll(lurkProgram, "\n", "")
		lurkProgram = strings.ReplaceAll(lurkProgram, "\t", "")
		assert.NoError(t, err)
		assert.Truef(t, isValid(lurkProgram), "Test %d should be valid", i)
		assert.Equalf(t, test.expected, lurkProgram, "Test %d not as expected", i)
	}
}

func TestCircularImports(t *testing.T) {
	mod1 := `!(module math (
			!(import utils)
			!(defun plus-three (x) (+ x 3))
		))

		!(module time (
			!(assert (<= !(param locktime-precision) 30))
		))
		`

	mod2 := `!(module utils (
			!(import math)
		))
		`

	tempDir := path.Join(os.TempDir(), "circular_import_test")
	defer os.Remove(tempDir)

	err := os.MkdirAll(tempDir, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "mod1.lurk"), []byte(mod1), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "mod2.lurk"), []byte(mod2), 0644)
	assert.NoError(t, err)

	mp, err := macros.NewMacroPreprocessor(macros.DependencyDir(tempDir))
	assert.NoError(t, err)

	lurkProgram := `!(defun my-func (y) (
				!(import math)
				(plus-two 10)
			))`

	_, err = mp.Preprocess(lurkProgram)
	assert.Error(t, err)
	fmt.Println(err)
	assert.True(t, errors.Is(err, macros.ErrCircularImports))
}

func TestWithStandardLib(t *testing.T) {
	mp, err := macros.NewMacroPreprocessor(macros.WithStandardLib(), macros.RemoveComments())
	assert.NoError(t, err)

	lurkProgram := `!(defun my-func (y) (
				!(import std/crypto)
				(check-sig 10)
			))`
	lurkProgram, err = mp.Preprocess(lurkProgram)
	assert.NoError(t, err)
	lurkProgram = strings.ReplaceAll(lurkProgram, "\n", "")
	lurkProgram = strings.ReplaceAll(lurkProgram, "\t", "")
	assert.True(t, isValid(lurkProgram))
	expected := `(letrec ((my-func (lambda (y) (        (letrec ((check-sig (lambda (signature pubkey sighash) (                (eval (cons 'check_ecc_sig (cons (car signature) (cons (car (cdr signature)) (cons (car pubkey) (cons (car (cdr pubkey)) (cons sighash nil)))))))        ))))(check-sig 10)))))))`
	assert.Equal(t, expected, lurkProgram)
}
