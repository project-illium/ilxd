// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"strings"
)

var paramMap = map[string]string{
	"txo-root":           "(list-get 1 public-params)",
	"fee":                "(list-get 2 public-params)",
	"coinbase":           "(list-get 3 public-params)",
	"mint-id":            "(list-get 4 public-params)",
	"mint-amount":        "(list-get 5 public-params)",
	"sighash":            "(list-get 7 public-params)",
	"locktime":           "(list-get 8 public-params)",
	"locktime-precision": "(list-get 9 public-params)",
}

var inputMap = map[string]int{
	"script-commitment":            0,
	"amount":                       1,
	"asset-id":                     2,
	"script-params":                3,
	"commitment-index":             4,
	"state":                        5,
	"salt":                         6,
	"unlocking-params":             7,
	"inclusion-proof-hashes":       8,
	"inclusion-proof-acccumulator": 9,
}

var outputMap = map[string]int{
	"script-commitment": 0,
	"amount":            1,
	"asset-id":          2,
	"state":             3,
	"salt":              4,
}

var pubOutMap = map[string]int{
	"commitment": 0,
	"ciphertext": 1,
}

func macroExpandParam(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(param") {
			p.pos += 8 // Skip over "!(param"
			paramStart := p.pos

			for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
				p.Consume()
			}
			paramName := p.input[paramStart:p.pos]

			if paramName == "nullifiers" {
				// Skip over potential whitespace
				for p.Peek() == ' ' {
					p.Consume()
				}
				indexStart := p.pos
				for p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				index := p.input[indexStart:p.pos]
				result += fmt.Sprintf("(list-get %s (list-get 0 public-params))", index)
			} else if paramName == "priv-in" {
				// Skip over potential whitespace
				for p.Peek() == ' ' {
					p.Consume()
				}
				indexStart := p.pos
				for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				index := p.input[indexStart:p.pos]
				resultExp := fmt.Sprintf("(list-get %s (list-get 0 public-params))", index)

				if p.Peek() == ' ' {
					// Consume whitespace and then check for sub-param
					p.Consume()
					subParamStart := p.pos
					for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
						p.Consume()
					}
					subParam := p.input[subParamStart:p.pos]
					if subIndex, ok := inputMap[subParam]; ok {
						result += fmt.Sprintf("(list-get %d %s)", subIndex, resultExp)
					} else {
						result += resultExp
					}
				} else {
					result += resultExp
				}

			} else if paramName == "priv-out" {
				// Skip over potential whitespace
				for p.Peek() == ' ' {
					p.Consume()
				}
				indexStart := p.pos
				for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				index := p.input[indexStart:p.pos]
				resultExp := fmt.Sprintf("(list-get %s (car (cdr private-params)))", index)

				if p.Peek() == ' ' {
					// Consume whitespace and then check for sub-param
					p.Consume()
					subParamStart := p.pos
					for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
						p.Consume()
					}
					subParam := p.input[subParamStart:p.pos]
					if subIndex, ok := outputMap[subParam]; ok {
						result += fmt.Sprintf("(list-get %d %s)", subIndex, resultExp)
					} else {
						result += resultExp
					}
				} else {
					result += resultExp
				}
			} else if paramName == "pub-out" {
				// Skip over potential whitespace
				for p.Peek() == ' ' {
					p.Consume()
				}
				indexStart := p.pos
				for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				index := p.input[indexStart:p.pos]
				resultExp := fmt.Sprintf("(list-get %s (list-get 6 public-params))", index)

				if p.Peek() == ' ' {
					// Consume whitespace and then check for sub-param
					p.Consume()
					subParamStart := p.pos
					for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
						p.Consume()
					}
					subParam := p.input[subParamStart:p.pos]
					if subIndex, ok := pubOutMap[subParam]; ok {
						result += fmt.Sprintf("(list-get %d %s)", subIndex, resultExp)
					} else {
						result += resultExp
					}
				} else {
					result += resultExp
				}
			} else if substitution, found := paramMap[paramName]; found {
				result += substitution
			} else {
				// In case the paramName is not found, let's just keep the original code
				result += "!(param" + paramName + ")"
			}

			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the param body
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

func macroExpandList(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(list") {
			p.pos += 7 // Skip over "!(list"
			var elements []string

			// Ensure we capture all elements and that we don't accidentally consume the closing parenthesis of !(list ... )
			for p.Peek() != ')' && p.Peek() != 0 {
				// Skip over potential whitespace
				for p.Peek() == ' ' {
					p.Consume()
				}
				var body string
				if p.Peek() == '(' {
					body = p.ParseSExpr() // Parse the s-expression if body starts with (
				} else {
					bodyStart := p.pos
					for p.Peek() != ' ' && p.Peek() != ')' && p.Peek() != 0 {
						p.Consume()
					}
					body = p.input[bodyStart:p.pos]
				}

				elements = append(elements, body)
			}

			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the list body

			if len(elements) > 0 {
				result += buildConsList(elements)
			} else {
				result += "nil"
			}
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

// Recursively builds a cons list from the elements
func buildConsList(elems []string) string {
	if len(elems) == 0 {
		return "nil"
	}
	if len(elems) == 1 {
		return fmt.Sprintf("(cons %s nil)", elems[0])
	}

	return fmt.Sprintf("(cons %s %s)", elems[0], buildConsList(elems[1:]))
}

func macroExpandAssert(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(assert") &&
			!strings.HasPrefix(p.input[p.pos:], "!(assert-eq") {
			p.pos += 9 // Skip over "!(assert"
			var body string
			if p.Peek() == '(' {
				body = p.ParseSExpr() // Parse the s-expression if body starts with (
			} else {
				bodyStart := p.pos
				for p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				body = p.input[bodyStart:p.pos]
			}
			result += fmt.Sprintf("(if (eq %s nil) nil", body)
			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the assert body
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

func macroExpandAssertEq(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(assert-eq") {
			p.pos += 12            // Skip over "!(assert-eq"
			val1 := p.ParseSExpr() // Parse the first value/expression

			// Skip over potential whitespace
			for p.Peek() == ' ' {
				p.Consume()
			}

			val2 := p.ParseSExpr() // Parse the second value/expression

			result += fmt.Sprintf("(if (eq (eq %s %s) nil) nil", val1, val2)
			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the assert-eq body
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

func macroExpandDef(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(def") &&
			!strings.HasPrefix(p.input[p.pos:], "!(defrec") &&
			!strings.HasPrefix(p.input[p.pos:], "!(defun") {
			p.pos += 6 // Skip over "!(def"
			variableName := strings.TrimSpace(p.ReadUntil(' '))
			p.Consume()
			var body string
			if p.Peek() == '(' {
				body = p.ParseSExpr() // Parse the s-expression if body starts with (
			} else {
				bodyStart := p.pos
				for p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				body = p.input[bodyStart:p.pos]
			}
			result += fmt.Sprintf("(let ((%s %s))", variableName, body)
			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the def body
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

func macroExpandDefrec(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(defrec") {
			p.pos += 9 // Skip over "!(defrec"
			variableName := strings.TrimSpace(p.ReadUntil(' '))
			p.Consume()
			var body string
			if p.Peek() == '(' {
				body = p.ParseSExpr() // Parse the s-expression if body starts with (
			} else {
				bodyStart := p.pos
				for p.Peek() != ')' && p.Peek() != 0 {
					p.Consume()
				}
				body = p.input[bodyStart:p.pos]
			}
			result += fmt.Sprintf("(letrec ((%s %s))", variableName, body)
			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the defrec body
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

func macroExpandDefun(lurkProgram string) string {
	p := NewParser(lurkProgram)
	result := ""

	for p.Peek() != 0 {
		if strings.HasPrefix(p.input[p.pos:], "!(defun") {
			p.pos += 8 // Skip over "!(defun"
			name := strings.TrimSpace(p.ReadUntil('('))
			fmt.Println(name)
			params := p.ParseSExpr()
			fmt.Println(params)
			p.ReadUntil('(')
			body := p.ParseSExpr()
			fmt.Println(body)

			result += fmt.Sprintf("(letrec ((%s (lambda %s %s)))", name, params, body)
			p.ReadUntil(')')
			p.Consume() // Consume the closing parenthesis after the defun body
		} else {
			result += string(p.Consume())
		}
	}
	return result
}

// PreProcess takes a lurk program string and expands all the macros.
func PreProcess(lurkProgram string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(lurkProgram))

	var (
		openCount      = 0
		parenthesesMap = make(map[int]int)
		modifiedLines  []string
	)

	for scanner.Scan() {
		line := scanner.Text()
		var modifiedLine strings.Builder
		for i, char := range line {
			modifiedLine.WriteRune(char)
			if char == '(' {
				openCount++
			} else if char == ')' {
				openCount--
				for c, p := range parenthesesMap {
					if c == openCount {
						for i := 0; i < p; i++ {
							modifiedLine.WriteRune(')')
						}
						delete(parenthesesMap, c)
					}
				}
			} else if char == '!' {
				if macro, ok := IsMacro(line[i:]); ok && macro.IsNested() {
					parenthesesMap[openCount-1]++
				}
			}
		}
		modifiedLines = append(modifiedLines, modifiedLine.String())
	}
	var modifiedLine strings.Builder
	for c, p := range parenthesesMap {
		if c == -1 {
			for i := 0; i < p; i++ {
				modifiedLine.WriteRune(')')
			}
			delete(parenthesesMap, c)
		}
	}
	modifiedLines = append(modifiedLines, modifiedLine.String())
	lurkProgram = strings.Join(modifiedLines, "\n")

	if err := scanner.Err(); err != nil {
		return "", err
	}

	for _, macro := range []Macro{Def, Defrec, Defun, Assert, AssertEq, List, Param} {
		lurkProgram = macro.Expand(lurkProgram)
	}

	return lurkProgram, nil
}
