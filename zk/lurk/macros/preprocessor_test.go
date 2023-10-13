// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros_test

import (
	"container/list"
	"fmt"
	"github.com/project-illium/ilxd/zk/lurk/macros"
	"github.com/stretchr/testify/assert"
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
		{"(+ x 3)", "(+ x 3)\n"},
		{"!(def x 3)", "(let ((x 3))\n)"},
		{"!(def x (car y))", "(let ((x (car y)))\n)"},
		{"!(def x 3) t", "(let ((x 3)) t\n)"},
		{"!(def x (car y)) t", "(let ((x (car y))) t\n)"},
		{"!(defrec x 3)", "(letrec ((x 3))\n)"},
		{"!(defrec x (car y))", "(letrec ((x (car y)))\n)"},
		{"!(defrec x 3) t", "(letrec ((x 3)) t\n)"},
		{"!(defrec x (car y)) t", "(letrec ((x (car y))) t\n)"},
		{"!(defun f (x) (+ x 3))", "(letrec ((f (lambda (x) (+ x 3))))\n)"},
		{"!(defun f (x) (+ x 3)) t", "(letrec ((f (lambda (x) (+ x 3)))) t\n)"},
		{"!(assert t)", "(if (eq t nil) nil\n)"},
		{"!(assert (+ x 5)) nil", "(if (eq (+ x 5) nil) nil nil\n)"},
		{"!(assert t) nil", "(if (eq t nil) nil nil\n)"},
		{"!(assert-eq x 3)", "(if (eq (eq x 3 ) nil) nil\n)"},
		{"!(assert-eq x 3) t", "(if (eq (eq x 3 ) nil) nil t\n)"},
		{"!(defun f (x) (!(assert t) 3))", "(letrec ((f (lambda (x) ((if (eq t nil) nil 3)))))\n)"},
		{"(lambda (script-params unlocking-params input-index private-params public-params)\n !(assert-eq (+ x 5) 4) !(def z 5) !(assert t) t)", "(lambda (script-params unlocking-params input-index private-params public-params)\n (if (eq (eq (+ x 5) 4) nil) nil (let ((z 5)) (if (eq t nil) nil t))))\n"},
		{"!(list 1 2 3 4)", "(cons 1 (cons 2 (cons 3 (cons 4 nil))))\n"},
		{"!(list 1 (car x) 3 4)", "(cons 1 (cons (car x) (cons 3 (cons 4 nil))))\n"},
		{"!(param nullifiers 0)", "(list-get 0 (list-get 0 public-params))\n"},
		{"!(param txo-root)", "(list-get 1 public-params)\n"},
		{"!(param fee)", "(list-get 2 public-params)\n"},
		{"!(param coinbase)", "(list-get 3 public-params)\n"},
		{"!(param mint-id)", "(list-get 4 public-params)\n"},
		{"!(param mint-amount)", "(list-get 5 public-params)\n"},
		{"!(param sighash)", "(list-get 7 public-params)\n"},
		{"!(param locktime)", "(list-get 8 public-params)\n"},
		{"!(param locktime-precision)", "(list-get 9 public-params)\n"},
		{"!(param priv-in 2)", "(list-get 2 (car private-params))\n"},
		{"!(param priv-out 3)", "(list-get 3 (car (cdr private-params)))\n"},
		{"!(param pub-out 4)", "(list-get 4 (list-get 6 public-params))\n"},
		{"!(param priv-in 2 script-commitment)", "(list-get 0 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 amount)", "(list-get 1 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 asset-id)", "(list-get 2 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 script-params)", "(list-get 3 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 commitment-index)", "(list-get 4 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 state)", "(list-get 5 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 salt)", "(list-get 6 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 unlocking-params)", "(list-get 7 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 inclusion-proof-hashes)", "(list-get 8 (list-get 2 (car private-params)))\n"},
		{"!(param priv-in 2 inclusion-proof-accumulator)", "(list-get 9 (list-get 2 (car private-params)))\n"},
		{"!(param priv-out 3 script-hash)", "(list-get 0 (list-get 3 (car (cdr private-params))))\n"},
		{"!(param priv-out 3 amount)", "(list-get 1 (list-get 3 (car (cdr private-params))))\n"},
		{"!(param priv-out 3 asset-id)", "(list-get 2 (list-get 3 (car (cdr private-params))))\n"},
		{"!(param priv-out 3 state)", "(list-get 3 (list-get 3 (car (cdr private-params))))\n"},
		{"!(param priv-out 3 salt)", "(list-get 4 (list-get 3 (car (cdr private-params))))\n"},
		{"!(param pub-out 4 commitment)", "(list-get 0 (list-get 4 (list-get 6 public-params)))\n"},
		{"!(param pub-out 4 ciphertext)", "(list-get 1 (list-get 4 (list-get 6 public-params)))\n"},
	}

	for i, test := range tests {
		lurkProgram, err := macros.PreProcess(test.input)
		assert.NoError(t, err)
		assert.Truef(t, isValid(lurkProgram), "Test %d should be valid", i)
		assert.Equalf(t, test.expected, lurkProgram, "Test %d not as expected", i)
	}
}
