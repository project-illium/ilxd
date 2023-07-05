package types

import (
	"testing"
)

const (
	testSerializedID = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
)

func TestNewIDFromString(t *testing.T) {
	id, err := NewIDFromString(testSerializedID)
	if err != nil {
		t.Error(err)
	}

	if id.String() != testSerializedID {
		t.Errorf("Expected %s, got %s", testSerializedID, id.String())
	}
}

func TestIDUnmarshalJSON(t *testing.T) {
	var id ID
	err := id.UnmarshalJSON([]byte(testSerializedID))
	if err != nil {
		t.Error(err)
	}

	if id.String() != testSerializedID {
		t.Errorf("Expected %s, got %s", testSerializedID, id.String())
	}
}
