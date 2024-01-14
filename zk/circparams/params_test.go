// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package circparams

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

func TestStandardPublicParams_ToExpr(t *testing.T) {
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
	s := StandardPublicParams{
		Nullifiers:        n,
		TXORoot:           types.NewID(rnd.random()),
		Fee:               11111,
		Outputs:           o,
		SigHash:           types.NewID(rnd.random()),
		Locktime:          time.Time{},
		LocktimePrecision: 600 * time.Second,
	}

	expr, err := s.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons 0x3e04cd7c36706f4745b50d96e5dea42e4fd60059defe5a8c02473326eb2d90a6 (cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e nil)))) (cons 0x2148dc86d6dd54404204768c213e12ed6d1d0d9ea867d6a249eb5bbb98705a2f (cons 11111 (cons nil (cons 0 (cons (cons (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons 0x17379becc7f415b7880349abe306e0f9a778317d88cedb923735d0be32268d0d (cons 0x209cd61fca4bbbf52c43bc981659f8693999b6c16f840c6f2b29c7ec17ca5de9 (cons 0x241f nil))))))))) nil) (cons -62135596800 (cons 600 nil)))))))))`
	assert.Equal(t, expected, expr)
}

func TestStandardPrivateParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

	priv := StandardPrivateParams{
		Inputs: []PrivateInput{
			{
				Amount:          12345,
				AssetID:         types.NewID(rnd.random()),
				State:           [][]byte{rnd.random(), []byte{0x00, 0x01}},
				Salt:            types.NewID(rnd.random()),
				CommitmentIndex: 0,
				InclusionProof: InclusionProof{
					Hashes: [][]byte{rnd.random(), rnd.random(), rnd.random(), rnd.random()},
					Flags:  3,
				},
				Script:          "(lambda (locking-params unlocking-params input-index private-params public-params) t)",
				LockingParams:   [][]byte{rnd.random(), rnd.random()},
				UnlockingParams: "(cons 1 (cons 2 (cons 3 nil)))",
			},
			{
				Amount:          777777,
				AssetID:         types.NewID(rnd.random()),
				State:           [][]byte{rnd.random(), []byte{0xff, 0xee, 0xdd}},
				Salt:            types.NewID(rnd.random()),
				CommitmentIndex: 1,
				InclusionProof: InclusionProof{
					Hashes: [][]byte{rnd.random(), rnd.random(), rnd.random(), rnd.random()},
					Flags:  7,
				},
				Script:          "(lambda (locking-params unlocking-params input-index private-params public-params) nil)",
				LockingParams:   [][]byte{rnd.random(), rnd.random()},
				UnlockingParams: "(cons 4 (cons 5 (cons 6 nil)))",
			},
		},
		Outputs: []PrivateOutput{
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     999999,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     65432,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
		},
	}

	expr, err := priv.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons (cons (cons 12345 (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 1 nil)) (cons 0 (cons (cons (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e t) (cons (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee t) (cons (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 nil) (cons (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e nil) nil)))) (cons (lambda (locking-params unlocking-params input-index private-params public-params) t) (cons (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 nil)) (cons (cons 1 (cons 2 (cons 3 nil))) nil))))))))) (cons (cons 777777 (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons 0x209cd61fca4bbbf52c43bc981659f8693999b6c16f840c6f2b29c7ec17ca5de9 (cons (cons 0x17379becc7f415b7880349abe306e0f9a778317d88cedb923735d0be32268d0d (cons 16772829 nil)) (cons 1 (cons (cons (cons 0x241f7f100e7018c3770d2cc366df2169a42aff68f12589ef40ced062453bfbfc t) (cons (cons 0x2148dc86d6dd54404204768c213e12ed6d1d0d9ea867d6a249eb5bbb98705a2f t) (cons (cons 0x3e04cd7c36706f4745b50d96e5dea42e4fd60059defe5a8c02473326eb2d90a6 t) (cons (cons 0x20a75a29803246715c3f1171d480575125fa03674a4b38c7392afd68209d2da5 nil) nil)))) (cons (lambda (locking-params unlocking-params input-index private-params public-params) nil) (cons (cons 0x10ff749d95d8b00723a4df5c32235883ec718e9cb77f4d281098083e39e0ec11 (cons 0x176cfd26394d0d8f2b4cb97d997ba38f68c913d9c96772ad7ab4c080ef970087 nil)) (cons (cons 4 (cons 5 (cons 6 nil))) nil))))))))) nil)) (cons (cons 0x388f9e0f3852bab162dbb7b1905b46501329427278e08c5fcd958d95c070a517 (cons 999999 (cons 0x17591b84c1d2cac02b68beb4b57be9d8f588e0632ec014061fb9b066fba1d156 (cons 0x3d57c4e7653d6e16eb887a6767bdd6a919ef2a47ac662c51d995d03b91ced0c3 (cons (cons 0x0870a663bed0366f7e09d89d0a90ddf028afb818001034f00e1f309cf75443d3 (cons 0x1e800b8d031709fb149f81c6f700846990ad1b64ebc37242ad1261f6273e9dd6 nil)) nil))))) (cons (cons 0x07b315ad63b427aede35c4cffb5a52175f4bd874ea86c8de1166f1824798d7d7 (cons 65432 (cons 0x39d7922c1d0e711d47c65f8d91622fb66ca191cd9180d1f64b187d1c9a1e9eba (cons 0x143aae07ea868a34da35d27e2edea7d563a37e2518d1211cdba7b9086d9926af (cons (cons 0x3b81c0a7cfb1a0609afee9858bf3199ef837599370f4fd7e52f25310d49bdc15 (cons 0x048939fe6aa8274156dbf88f9bfef3ec6d1b99142fd6a2bbaa4fb8951b7cf72f nil)) nil))))) nil)))`
	assert.Equal(t, expected, expr)
}

func TestMintPublicParams_ToExpr(t *testing.T) {
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
	mintID := rnd.random()
	s := MintPublicParams{
		Nullifiers:        n,
		TXORoot:           types.NewID(rnd.random()),
		Fee:               11111,
		Outputs:           o,
		SigHash:           types.NewID(rnd.random()),
		Locktime:          time.Time{},
		LocktimePrecision: 600 * time.Second,
		MintID:            types.NewID(mintID),
		MintAmount:        1234,
	}

	expr, err := s.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons 0x20a75a29803246715c3f1171d480575125fa03674a4b38c7392afd68209d2da5 (cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e nil)))) (cons 0x3e04cd7c36706f4745b50d96e5dea42e4fd60059defe5a8c02473326eb2d90a6 (cons 11111 (cons 0x2148dc86d6dd54404204768c213e12ed6d1d0d9ea867d6a249eb5bbb98705a2f (cons 1234 (cons (cons (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons 0x17379becc7f415b7880349abe306e0f9a778317d88cedb923735d0be32268d0d (cons 0x209cd61fca4bbbf52c43bc981659f8693999b6c16f840c6f2b29c7ec17ca5de9 (cons 0x241f nil))))))))) nil) (cons -62135596800 (cons 600 nil)))))))))`
	assert.Equal(t, expected, expr)
}

func TestMintPrivateParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

	priv := MintPrivateParams{
		Inputs: []PrivateInput{
			{
				Amount:          12345,
				AssetID:         types.NewID(rnd.random()),
				State:           [][]byte{rnd.random(), []byte{0x00, 0x01}},
				Salt:            types.NewID(rnd.random()),
				CommitmentIndex: 0,
				InclusionProof: InclusionProof{
					Hashes: [][]byte{rnd.random(), rnd.random(), rnd.random(), rnd.random()},
					Flags:  3,
				},
				Script:          "(lambda (locking-params unlocking-params input-index private-params public-params) t)",
				LockingParams:   [][]byte{rnd.random(), rnd.random()},
				UnlockingParams: "(cons 1 (cons 2 (cons 3 nil)))",
			},
			{
				Amount:          777777,
				AssetID:         types.NewID(rnd.random()),
				State:           [][]byte{rnd.random(), []byte{0xff, 0xee, 0xdd}},
				Salt:            types.NewID(rnd.random()),
				CommitmentIndex: 1,
				InclusionProof: InclusionProof{
					Hashes: [][]byte{rnd.random(), rnd.random(), rnd.random(), rnd.random()},
					Flags:  7,
				},
				Script:          "(lambda (locking-params unlocking-params input-index private-params public-params) nil)",
				LockingParams:   [][]byte{rnd.random(), rnd.random()},
				UnlockingParams: "(cons 4 (cons 5 (cons 6 nil)))",
			},
		},
		Outputs: []PrivateOutput{
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     999999,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     65432,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
		},
	}

	expr, err := priv.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons (cons (cons 12345 (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 1 nil)) (cons 0 (cons (cons (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e t) (cons (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee t) (cons (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 nil) (cons (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e nil) nil)))) (cons (lambda (locking-params unlocking-params input-index private-params public-params) t) (cons (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 nil)) (cons (cons 1 (cons 2 (cons 3 nil))) nil))))))))) (cons (cons 777777 (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons 0x209cd61fca4bbbf52c43bc981659f8693999b6c16f840c6f2b29c7ec17ca5de9 (cons (cons 0x17379becc7f415b7880349abe306e0f9a778317d88cedb923735d0be32268d0d (cons 16772829 nil)) (cons 1 (cons (cons (cons 0x241f7f100e7018c3770d2cc366df2169a42aff68f12589ef40ced062453bfbfc t) (cons (cons 0x2148dc86d6dd54404204768c213e12ed6d1d0d9ea867d6a249eb5bbb98705a2f t) (cons (cons 0x3e04cd7c36706f4745b50d96e5dea42e4fd60059defe5a8c02473326eb2d90a6 t) (cons (cons 0x20a75a29803246715c3f1171d480575125fa03674a4b38c7392afd68209d2da5 nil) nil)))) (cons (lambda (locking-params unlocking-params input-index private-params public-params) nil) (cons (cons 0x10ff749d95d8b00723a4df5c32235883ec718e9cb77f4d281098083e39e0ec11 (cons 0x176cfd26394d0d8f2b4cb97d997ba38f68c913d9c96772ad7ab4c080ef970087 nil)) (cons (cons 4 (cons 5 (cons 6 nil))) nil))))))))) nil)) (cons (cons 0x388f9e0f3852bab162dbb7b1905b46501329427278e08c5fcd958d95c070a517 (cons 999999 (cons 0x17591b84c1d2cac02b68beb4b57be9d8f588e0632ec014061fb9b066fba1d156 (cons 0x3d57c4e7653d6e16eb887a6767bdd6a919ef2a47ac662c51d995d03b91ced0c3 (cons (cons 0x0870a663bed0366f7e09d89d0a90ddf028afb818001034f00e1f309cf75443d3 (cons 0x1e800b8d031709fb149f81c6f700846990ad1b64ebc37242ad1261f6273e9dd6 nil)) nil))))) (cons (cons 0x07b315ad63b427aede35c4cffb5a52175f4bd874ea86c8de1166f1824798d7d7 (cons 65432 (cons 0x39d7922c1d0e711d47c65f8d91622fb66ca191cd9180d1f64b187d1c9a1e9eba (cons 0x143aae07ea868a34da35d27e2edea7d563a37e2518d1211cdba7b9086d9926af (cons (cons 0x3b81c0a7cfb1a0609afee9858bf3199ef837599370f4fd7e52f25310d49bdc15 (cons 0x048939fe6aa8274156dbf88f9bfef3ec6d1b99142fd6a2bbaa4fb8951b7cf72f nil)) nil))))) nil)))`
	assert.Equal(t, expected, expr)
}

func TestStakePublicParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

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

	s := StakePublicParams{
		StakeAmount: 333333,
		SigHash:     types.NewID(rnd.random()),
		Nullifier:   types.NewNullifier(rnd.random()),
		TXORoot:     types.NewID(rnd.random()),
		LockedUntil: time.Unix(54321, 0),
	}

	expr, err := s.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons 333333 (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons (cons 0x17379becc7f415b7880349abe306e0f9a778317d88cedb923735d0be32268d0d nil) (cons 0x209cd61fca4bbbf52c43bc981659f8693999b6c16f840c6f2b29c7ec17ca5de9 (cons 0 (cons nil (cons 0 (cons nil  (cons 54321 (cons 0 nil))))))))))`
	assert.Equal(t, expected, expr)
}

func TestStakePrivateParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

	priv := StakePrivateParams(
		PrivateInput{
			Amount:          12345,
			AssetID:         types.NewID(rnd.random()),
			State:           [][]byte{rnd.random(), []byte{0x00, 0x01}},
			Salt:            types.NewID(rnd.random()),
			CommitmentIndex: 0,
			InclusionProof: InclusionProof{
				Hashes: [][]byte{rnd.random(), rnd.random(), rnd.random(), rnd.random()},
				Flags:  3,
			},
			Script:          "(lambda (locking-params unlocking-params input-index private-params public-params) t)",
			LockingParams:   [][]byte{rnd.random(), rnd.random()},
			UnlockingParams: "(cons 1 (cons 2 (cons 3 nil)))",
		})

	expr, err := priv.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons 12345 (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 1 nil)) (cons 0 (cons (cons (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e t) (cons (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee t) (cons (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 nil) (cons (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e nil) nil)))) (cons (lambda (locking-params unlocking-params input-index private-params public-params) t) (cons (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 nil)) (cons (cons 1 (cons 2 (cons 3 nil))) nil)))))))))`
	assert.Equal(t, expected, expr)
}

func TestCoinbasePublicParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

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
	s := CoinbasePublicParams{
		Outputs:  o,
		Coinbase: 98765,
	}

	expr, err := s.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons 98765 (cons (cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f nil))))))))) nil) nil))`
	assert.Equal(t, expected, expr)
}

func TestCoinbasePrivateParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

	priv := CoinbasePrivateParams(
		[]PrivateOutput{
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     999999,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     65432,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
		},
	)

	expr, err := priv.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 999999 (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e nil)) nil))))) (cons (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 65432 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 nil)) nil))))) nil))`
	assert.Equal(t, expected, expr)
}

func TestTreasuryPublicParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

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
	s := TreasuryPublicParams{
		Outputs:                o,
		TreasuryWithdrawAmount: 98765,
	}

	expr, err := s.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons 98765 (cons (cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f nil))))))))) nil) nil))`
	assert.Equal(t, expected, expr)
}

func TestTreasuryPrivateParams_ToExpr(t *testing.T) {
	rnd := deterministicRand{seed: 0}

	priv := TreasuryPrivateParams(
		[]PrivateOutput{
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     999999,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
			{
				ScriptHash: types.NewID(rnd.random()),
				Amount:     65432,
				AssetID:    types.NewID(rnd.random()),
				State:      [][]byte{rnd.random(), rnd.random()},
				Salt:       types.NewID(rnd.random()),
			},
		},
	)

	expr, err := priv.ToExpr()
	assert.NoError(t, err)
	assert.True(t, macros.IsValidLurk(expr))

	expected := `(cons (cons 0x320b5ea99e653bc2b593db4130d10a4efd3a0b4cc2e1a6672b678d71dfbd33ad (cons 999999 (cons 0x143aa29cb26d5a16e077395c7760432bdef05afeee6327f53df8217d1502dc63 (cons 0x123904eb31391a352f7a15f8fb8215394a4459296ba0a556bf925490452145ee (cons (cons 0x2fb047a7e9f84cb45b459c5cee3a54bb1b69ce73cd2b3a0b3794a0a875eb0618 (cons 0x15e1d57429e0f629485fbb3ef63e055552cf660ba7edfc56bcaa52ab51bdf71e nil)) nil))))) (cons (cons 0x0c7ad0c8599fe9958d0d3b8aa05c39786e8031addfba483bc6c6aa3d8fe7eb06 (cons 65432 (cons 0x326e0390e49d10c80cc1450568d47e28329b774864b79ab48336a5111a8a893e (cons 0x258f7bccd5548075c0fb752ff19abb7dd1ee5c1d157c3fa93d125c452b743124 (cons (cons 0x07832d56ed983f74231e25768a2e4a46d1a6e108727a53b861902bb2dfe3ca62 (cons 0x026f215279fa4fcf0c68379a0e2bb2f716d616e95169a0ccc4920e7970020587 nil)) nil))))) nil))`
	assert.Equal(t, expected, expr)
}
