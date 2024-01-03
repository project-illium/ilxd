package zk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTagFromBytes(t *testing.T) {
	tags := []Tag{
		TagNil,
		TagCons,
		TagSym,
		TagFun,
		TagNum,
		TagThunk,
		TagStr,
		TagChar,
		TagComm,
		TagU64,
		TagKey,
		TagCproc,
	}

	for _, tag := range tags {
		b := make([]byte, 32)
		b[31] = byte(tag)

		ret, err := TagFromBytes(b)
		assert.NoError(t, err)
		assert.Equal(t, tag, ret)
	}
}
