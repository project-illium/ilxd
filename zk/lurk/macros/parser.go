// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros

import "strings"

type Parser struct {
	input  string
	pos    int
	length int
}

func NewParser(input string) *Parser {
	return &Parser{
		input:  input,
		pos:    0,
		length: len(input),
	}
}

func (p *Parser) Peek() byte {
	if p.pos >= p.length {
		return 0
	}
	return p.input[p.pos]
}

func (p *Parser) Consume() byte {
	char := p.Peek()
	p.pos++
	return char
}

func (p *Parser) ReadUntil(c byte) string {
	start := p.pos
	for p.Peek() != c && p.Peek() != 0 {
		p.Consume()
	}
	return p.input[start:p.pos]
}

func (p *Parser) ParseSExpr() string {
	var result strings.Builder
	result.WriteByte(p.Consume()) // Consume opening (
	for p.Peek() != 0 {
		if p.Peek() == '(' {
			result.WriteString(p.ParseSExpr())
		} else if p.Peek() == ')' {
			result.WriteByte(p.Consume()) // Consume closing )
			return result.String()
		} else {
			result.WriteByte(p.Consume())
		}
	}
	return result.String()
}
