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

type UnlockingScript struct {
	ScriptCommitment []byte
	ScriptParams     [][]byte
}

func (u *UnlockingScript) Serialize() []byte {
	ser := make([]byte, len(u.ScriptCommitment))
	copy(ser, u.ScriptCommitment)
	for _, param := range u.ScriptParams {
		p := make([]byte, len(param)+1)
		copy(p[1:], param)
		p[0] = byte(len(p) - 1)
		ser = append(ser, p...)
	}
	return ser
}

func (ul *UnlockingScript) Deserialize(ser []byte) error {
	ul.ScriptCommitment = make([]byte, zk.CommitmentLen)
	copy(ul.ScriptCommitment, ser[0:zk.CommitmentLen])

	params, err := parseByteSlice(ser[zk.CommitmentLen:])
	if err != nil {
		return err
	}
	ul.ScriptParams = params
	return nil
}

func (u *UnlockingScript) lurkExpression() (string, error) {
	expr := fmt.Sprintf("(cons 0x%x ", u.ScriptCommitment)
	for _, param := range u.ScriptParams {
		if len(param) > 32 {
			return "", errors.New("script param exceeds max size")
		}
		if len(param) == 32 {
			expr += fmt.Sprintf("(cons 0x%x ", param)
		} else if len(param) <= 8 && len(param) > 0 {
			for {
				if len(param) == 8 {
					break
				}
				param = append([]byte{0x00}, param...)
			}
			n := binary.BigEndian.Uint64(param)
			expr += fmt.Sprintf("(cons %d ", n)
		} else if param == nil {
			expr += "(cons nil "
		} else {
			i := new(big.Int).SetBytes(param)
			expr += fmt.Sprintf("(cons %s ", i.String())
		}
	}
	expr += "nil)"
	for i := 0; i < len(u.ScriptParams); i++ {
		expr += ")"
	}
	return expr, nil
}

func (u *UnlockingScript) Hash() (ID, error) {
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

// SpendNote holds all the data that makes up an output commitment.
type SpendNote struct {
	ScriptHash []byte
	Amount     Amount
	AssetID    ID
	State      [StateLen]byte
	Salt       [SaltLen]byte
}

// Commitment serializes and hashes the data in the note and
// returns the hash.
func (s *SpendNote) Commitment() ID {
	ser := s.Serialize()
	return NewIDFromData(ser)
}

func (s *SpendNote) Serialize() []byte {
	ser := make([]byte, 0, ScriptHashLen+AmountLen+AssetIDLen+StateLen+SaltLen)

	idBytes := s.AssetID.Bytes()
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, uint64(s.Amount))

	ser = append(ser, s.ScriptHash...)
	ser = append(ser, amountBytes...)
	ser = append(ser, idBytes...)
	ser = append(ser, s.State[:]...)
	ser = append(ser, s.Salt[:]...)
	return ser
}

func (s *SpendNote) Deserialize(ser []byte) error {
	if len(ser) != ScriptHashLen+AmountLen+AssetIDLen+StateLen+SaltLen {
		return errors.New("invalid serialization length")
	}
	s.ScriptHash = make([]byte, ScriptHashLen)
	copy(s.ScriptHash, ser[:ScriptHashLen])
	s.Amount = Amount(binary.BigEndian.Uint64(ser[ScriptHashLen : ScriptHashLen+AmountLen]))
	copy(s.AssetID[:], ser[ScriptHashLen+AmountLen:ScriptHashLen+AmountLen+AssetIDLen])
	copy(s.State[:], ser[ScriptHashLen+AmountLen+AssetIDLen:ScriptHashLen+AmountLen+AssetIDLen+StateLen])
	copy(s.Salt[:], ser[ScriptHashLen+AmountLen+AssetIDLen+StateLen:])
	return nil
}

func parseByteSlice(data []byte) ([][]byte, error) {
	var result [][]byte
	i := 0

	for i < len(data) {
		// Read the length of the current element
		length := int(data[i])
		i++

		// Check for boundary issues
		if i+length > len(data) {
			// Handle error or break according to your requirement
			return nil, errors.New("invalid element size")
		}

		// Extract the element and add it to the result
		element := data[i : i+length]
		e := make([]byte, len(element))
		copy(e, element)

		result = append(result, e)
		i += length
	}

	return result, nil
}
