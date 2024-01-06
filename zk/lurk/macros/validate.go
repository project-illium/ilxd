// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros

import (
	"container/list"
	"fmt"
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

func IsValidLurk(s string) bool {
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
