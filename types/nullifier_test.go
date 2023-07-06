package types

import (
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
