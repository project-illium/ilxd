package zk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLurkEncrypt(t *testing.T) {
	plainText := make([][32]byte, 10)
	for i := 0; i < 10; i++ {
		data, err := randomFieldElement()
		assert.NoError(t, err)
		plainText[i] = data
	}

	key, err := randomFieldElement()
	assert.NoError(t, err)

	cipherText, err := LurkEncrypt(plainText, key)
	assert.NoError(t, err)

	plainText2, err := LurkDecrypt(cipherText, key)
	assert.NoError(t, err)

	for i := range plainText {
		assert.Equal(t, plainText[i], plainText2[i])
	}
}
