// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

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
)

func (m Macro) IsNested() bool {
	switch m {
	case Def, Defrec, Defun, Assert, AssertEq:
		return true
	default:
		return false
	}
}

func IsMacro(s string) (Macro, bool) {
	if strings.HasPrefix(s, "!(") {
		s = s[2:]
	}
	s = strings.ToLower(s)
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
