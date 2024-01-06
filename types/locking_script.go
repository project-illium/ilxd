// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/project-illium/ilxd/zk"
	"math/big"
)

var IlliumCoinID = NewID(bytes.Repeat([]byte{0x00}, 32))

const (
	CommitmentLen = 32
	ScriptHashLen = 32
	AmountLen     = 8
	AssetIDLen    = 32
	StateLen      = 128
	SaltLen       = 32
)

type LockingParams [][]byte

func (lp *LockingParams) ToExpr() (string, error) {
	if lp == nil || len(*lp) == 0 {
		return "nil", nil
	}
	return buildLurkExpression(*lp)
}

// LockingScript represents a utxo script in which coins are locked.
type LockingScript struct {
	ScriptCommitment ID
	LockingParams    [][]byte
}

// Serialize returns the locking script serialized as a byte slice
func (u *LockingScript) Serialize() []byte {
	ser := make([]byte, len(u.ScriptCommitment))
	copy(ser, u.ScriptCommitment[:])
	ser = append(ser, serializeData(u.LockingParams)...)
	return ser
}

// Deserialize turns a serialized byte slice back into an LockingScript
func (ul *LockingScript) Deserialize(ser []byte) error {
	copy(ul.ScriptCommitment[:], ser[0:zk.CommitmentLen])

	params, err := deserializeData(ser[zk.CommitmentLen:])
	if err != nil {
		return err
	}
	ul.LockingParams = params
	return nil
}

// Hash returns the Lurk Commitment hash of the locking script
func (u *LockingScript) Hash() (ID, error) {
	expr, err := u.lurkExpression()
	if err != nil {
		return ID{}, err
	}
	h, err := zk.LurkCommit(expr)
	if err != nil {
		return ID{}, err
	}
	return NewID(h), nil
}

func (u *LockingScript) lurkExpression() (string, error) {
	return buildLurkExpression(append([][]byte{u.ScriptCommitment[:]}, u.LockingParams...))
}

func serializeData(data [][]byte) []byte {
	var ser []byte
	for _, param := range data {
		p := make([]byte, len(param)+1)
		copy(p[1:], param)
		p[0] = byte(len(p) - 1)
		ser = append(ser, p...)
	}
	return ser
}

func deserializeData(ser []byte) ([][]byte, error) {
	var result [][]byte
	i := 0

	for i < len(ser) {
		if ser[i] == 0x00 {
			break
		}
		// Read the length of the current element
		length := int(ser[i])
		i++

		// Check for boundary issues
		if i+length > len(ser) {
			// Handle error or break according to your requirement
			return nil, errors.New("invalid element size")
		}

		// Extract the element and add it to the result
		element := ser[i : i+length]
		e := make([]byte, len(element))
		copy(e, element)

		result = append(result, e)
		i += length
	}

	return result, nil
}

func buildLurkExpression[T any](elems []T) (string, error) {
	expr := ""
	for _, elem := range elems {
		switch e := any(elem).(type) {
		case []byte:
			if len(e) > 32 {
				return "", errors.New("script param exceeds max size")
			}
			if len(e) == 32 {
				expr += fmt.Sprintf("(cons 0x%x ", e)
			} else if len(e) == 1 {
				if e[0] == 0x00 {
					expr += "(cons nil "
				} else {
					expr += "(cons t "
				}
			} else if len(e) <= 8 && len(e) > 1 {
				for i := len(e); i < 8; i++ {
					e = append([]byte{0x00}, e...)
				}
				n := binary.BigEndian.Uint64(e)
				expr += fmt.Sprintf("(cons %d ", n)
			} else if e == nil {
				expr += "(cons nil "
			} else {
				i := new(big.Int).SetBytes(e)
				expr += fmt.Sprintf("(cons %s ", i.String())
			}
		case string:
			expr += fmt.Sprintf("(cons %s ", e)
		default:
			return "", errors.New("unknown type")
		}
	}
	expr += "nil)"
	for i := 0; i < len(elems)-1; i++ {
		expr += ")"
	}
	return expr, nil
}
