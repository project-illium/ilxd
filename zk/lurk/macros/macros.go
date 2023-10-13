// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package macros

import "strings"

type Macro string

func (m Macro) String() string {
	return string(m)
}

const (
	Def      Macro = "def"
	Defrec   Macro = "defrec"
	Defun    Macro = "defun"
	List     Macro = "list"
	Param    Macro = "param"
	Assert   Macro = "assert"
	AssertEq Macro = "assert-eq"
	Import   Macro = "import"
)

func (m Macro) IsNested() bool {
	switch m {
	case Def, Defrec, Defun, Assert, AssertEq:
		return true
	default:
		return false
	}
}

func (m Macro) Expand(program string) string {
	switch m {
	case Def:
		return macroExpandDef(program)
	case Defrec:
		return macroExpandDefrec(program)
	case Defun:
		return macroExpandDefun(program)
	case Assert:
		return macroExpandAssert(program)
	case AssertEq:
		return macroExpandAssertEq(program)
	case List:
		return macroExpandList(program)
	case Param:
		return macroExpandParam(program)
	}
	return program
}

func IsMacro(s string) (Macro, bool) {
	s = strings.TrimPrefix(strings.ToLower(s), "!(")
	if strings.HasPrefix(s, Def.String()) {
		return Def, true
	} else if strings.HasPrefix(s, Defrec.String()) {
		return Defrec, true
	} else if strings.HasPrefix(s, Defun.String()) {
		return Defun, true
	} else if strings.HasPrefix(s, List.String()) {
		return List, true
	} else if strings.HasPrefix(s, Param.String()) {
		return Param, true
	} else if strings.HasPrefix(s, Assert.String()) {
		return Assert, true
	} else if strings.HasPrefix(s, AssertEq.String()) {
		return AssertEq, true
	}
	return "", false
}
