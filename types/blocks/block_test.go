package blocks

import (
	"testing"
)

func TestBlockHeaderUnmarshalJSON(t *testing.T) {
	serializedBlockHeader := `{"version": 1}`
	var bh BlockHeader
	err := bh.UnmarshalJSON([]byte(serializedBlockHeader))
	if err != nil {
		t.Fatal(err)
	}

	if bh.Version != 1 {
		t.Errorf("Expected %d, got %d", 1, bh.Version)
	}
}
