package types

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testSerializedNullifier = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
)

func TestNewNullifierFromString(t *testing.T) {
	n, err := NewNullifierFromString(testSerializedNullifier)
	if err != nil {
		t.Error(err)
	}

	if n.String() != testSerializedNullifier {
		t.Errorf("Expected %s, got %s", testSerializedNullifier, n.String())
	}
}

func TestNullifierUnmarshalJson(t *testing.T) {
	var n Nullifier
	err := n.UnmarshalJSON([]byte(testSerializedNullifier))
	if err != nil {
		t.Error(err)
	}

	if n.String() != testSerializedNullifier {
		t.Errorf("Expected %s, got %s", testSerializedNullifier, n.String())
	}
}

func TestCalculateNullifier(t *testing.T) {
	commitment, err := hex.DecodeString("0530365c951beb58cbd53df8441097165b5666853dc5a3610fbf605f6aa8ba52")
	assert.NoError(t, err)

	param1, err := hex.DecodeString("0890f5f7ed82055dad922130d72ef4b8764d7ff16a42d718ee4f60e974842932")
	assert.NoError(t, err)

	param2, err := hex.DecodeString("15741f17bbfa9513db079602cb53e05f3467da7633ac3852307cd341830558ca")
	assert.NoError(t, err)

	n, err := CalculateNullifier(123, [32]byte{}, commitment, param1, param2)
	assert.NoError(t, err)

	assert.Equal(t, "112c36d51636533954aef733108d223ab2e7d57623ac27e6805d21420c463155", n.String())
}
