package zk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
)

// LurkEncrypt is a stream cipher algorithm that can be computed inside
// the circuit. It operates on a lurk list where each item in the list is
// as lurk field element.
//
// This implementation operates outside the circuit as is used for computing
// an output's ciphertext field.
//
// There is a small probability that the ciphertext chunk may exceed the maximum
// field element resulting in a loss of one bit precision. To avoid this we brute
// force the nonce key to make sure no precision is lost.
//
// NOTE: for most transactions you want to use crypto/Encrypt which uses curve25519.
// This is only if you have a script where you need to verify ciphertext inside
// the script.
func LurkEncrypt(plaintext [][32]byte, key [32]byte) ([][32]byte, error) {
	ciphertext := make([][32]byte, len(plaintext)+1)
loop:
	for {
		nonce, err := randomFieldElement()
		if err != nil {
			return nil, err
		}
		for i, chunk := range plaintext {
			chunkKey, err := LurkCommit(fmt.Sprintf("(cons 0x%s (cons 0x%s (cons %d nil)))", hex.EncodeToString(key[:]), hex.EncodeToString(nonce[:]), i))
			if err != nil {
				return nil, err
			}
			encChunk := xorBytes(chunk[:], chunkKey)
			if encChunk[0]&0b11000000 > 0 {
				continue loop
			}
			copy(ciphertext[i+1][:], encChunk)
		}
		ciphertext[0] = nonce
		break
	}
	return ciphertext, nil
}

// LurkDecrypt decrypts the ciphertext using the key
func LurkDecrypt(ciphertext [][32]byte, key [32]byte) ([][32]byte, error) {
	plaintext := make([][32]byte, len(ciphertext)-1)
	nonce := ciphertext[0]
	for i, chunk := range ciphertext[1:] {
		chunkKey, err := LurkCommit(fmt.Sprintf("(cons 0x%s (cons 0x%s (cons %d nil)))", hex.EncodeToString(key[:]), hex.EncodeToString(nonce[:]), i))
		if err != nil {
			return nil, err
		}
		decChunk := xorBytes(chunk[:], chunkKey)
		copy(plaintext[i][:], decChunk)
	}
	return plaintext, nil
}

func xorBytes(a, b []byte) []byte {
	result := make([]byte, 32)
	for i := range a {
		result[i] = a[i] ^ b[i]
	}
	return result
}

func randomFieldElement() ([32]byte, error) {
	upperBound := new(big.Int)
	upperBound.SetString(LurkMaxFieldElement, 16)

	// Generate a random number in the range [0, upperBound)
	randomNum, err := rand.Int(rand.Reader, upperBound)
	if err != nil {
		return [32]byte{}, err
	}

	var ret [32]byte
	randomBytes := randomNum.Bytes()

	startIndex := len(ret) - len(randomBytes)
	copy(ret[startIndex:], randomBytes)
	return ret, nil
}
