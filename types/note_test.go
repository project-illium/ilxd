// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSpendNote_Hash(t *testing.T) {
	sh, err := hex.DecodeString("13e0143cceae5e7e44d8025c57f4759cfb6384e4a2d3d1106e6c098603845900")
	assert.NoError(t, err)
	param2, err := hex.DecodeString("0cef7dd85c04c505d55c063824a5bad62170db0d37e2068fc6c749ada2cb8293")
	assert.NoError(t, err)
	param1 := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	saltBytes, err := hex.DecodeString("0632779eccaeee683c5a15218e5d314d4f65f26b3087c87e256a5880defada0b")
	assert.NoError(t, err)
	var salt [32]byte
	copy(salt[:], saltBytes)

	note := &SpendNote{
		ScriptHash: NewID(sh),
		Amount:     12345,
		AssetID:    ID{},
		Salt:       salt,
		State:      State{param1, param2},
	}

	commitment, err := note.Commitment()
	assert.NoError(t, err)

	assert.Equal(t, "033936349e765fed388a814e2678533427fda4b9c785ff2383fced933128e418", commitment.String())

	s2, err := RandomSalt()
	assert.NoError(t, err)
	note2 := &SpendNote{
		ScriptHash: NewID(sh),
		Amount:     12345,
		AssetID:    ID{},
		Salt:       s2,
		State:      State{},
	}
	commitment2, err := note2.Commitment()
	assert.NoError(t, err)

	s3, err := RandomSalt()
	assert.NoError(t, err)
	note3 := &SpendNote{
		ScriptHash: NewID(sh),
		Amount:     12345,
		AssetID:    ID{},
		Salt:       s3,
		State:      State{},
	}
	commitment3, err := note3.Commitment()
	assert.NoError(t, err)

	assert.NotEqual(t, commitment2, commitment3)
}

func TestSpendNote_SerializeDeserialize(t *testing.T) {
	sh, err := hex.DecodeString("13e0143cceae5e7e44d8025c57f4759cfb6384e4a2d3d1106e6c098603845900")
	assert.NoError(t, err)
	param2, err := hex.DecodeString("0cef7dd85c04c505d55c063824a5bad62170db0d37e2068fc6c749ada2cb8293")
	assert.NoError(t, err)
	param1 := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	saltBytes, err := hex.DecodeString("0632779eccaeee683c5a15218e5d314d4f65f26b3087c87e256a5880defada0b")
	assert.NoError(t, err)
	var salt [32]byte
	copy(salt[:], saltBytes)

	note := &SpendNote{
		ScriptHash: NewID(sh),
		Amount:     12345,
		AssetID:    ID{},
		Salt:       salt,
		State:      State{param1, param2},
	}

	ser, err := note.Serialize()
	assert.NoError(t, err)

	note2 := new(SpendNote)
	err = note2.Deserialize(ser)
	assert.NoError(t, err)

	assert.Equal(t, note.ScriptHash, note2.ScriptHash)
	assert.Equal(t, note.Amount, note2.Amount)
	assert.Equal(t, note.AssetID, note2.AssetID)
	assert.Equal(t, note.Salt, note2.Salt)
	assert.Equal(t, note.State, note2.State)
}
