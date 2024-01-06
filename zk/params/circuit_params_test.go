// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"encoding/binary"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/zk/lurk/macros"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type deterministicRand struct {
	seed uint32
}

func (r *deterministicRand) random() []byte {
	b := make([]byte, 32)
	binary.BigEndian.PutUint32(b, r.seed)
	r.seed++
	return hash.HashFunc(b)
}

func TestPublicParamsToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

	n := make([]types.Nullifier, 0, 4)
	for i := 0; i < 4; i++ {
		r := rnd.random()
		n = append(n, types.NewNullifier(r))
	}
	o := make([]PublicOutput, 0, 1)
	for i := 0; i < 1; i++ {
		r := rnd.random()
		out := PublicOutput{
			Commitment: types.NewID(r[:]),
			CipherText: make([]byte, 0, 226),
		}
		for x := 0; x < 7; x++ {
			out.CipherText = append(out.CipherText, rnd.random()...)
		}
		out.CipherText = append(out.CipherText, rnd.random()[:2]...)
		o = append(o, out)
	}
	s := PublicParams{
		Nullifiers:        n,
		TXORoot:           types.NewID(rnd.random()),
		Fee:               11111,
		Coinbase:          22222,
		MintID:            types.NewID(rnd.random()),
		MintAmount:        33333,
		Outputs:           o,
		SigHash:           types.NewID(rnd.random()),
		Locktime:          time.Time{},
		LocktimePrecision: 600 * time.Second,
	}

	expr, err := s.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e nil)))) (cons 0x2148dc86d6dd54404204768c213e12ed6d1d0d9ea867d6a249eb5bbb98705a2f (cons 11111 (cons 22222 (cons 0x3e04cd7c36706f4745b50d96e5dea42e4fd60059defe5a8c02473326eb2d90a6 (cons 33333 (cons (cons (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons 0x17379becc7f415b7880349abe306e0f9a778317d88cedb923735d0be32268d0d (cons 0x209cd61fca4bbbf52c43bc981659f8693999b6c16f840c6f2b29c7ec17ca5de9 (cons 0x241f nil))))))))) nil) (cons 0x20a75a29803246715c3f1171d480575125fa03674a4b38c7392afd68209d2da5 (cons -62135596800 (cons 600 nil))))))))))`
	assert.Equal(t, expected, expr)
}
