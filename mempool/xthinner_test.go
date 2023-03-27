// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"encoding/hex"
	"fmt"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMempool_EncodeXthinner(t *testing.T) {
	blockHashes := []string{
		"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
		"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
		"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
	}

	mempoolHashes := []string{
		"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
		"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
		"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
		"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
		"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
	}

	// pops 0 0 1 0
	// pushes 1 0 0 0
	// pushbytes 0x25, 0x2f, 0x79 0xcd
	// stack 0xcd

	m := &Mempool{
		pool: make(map[types.ID]*transactions.Transaction),
	}
	for _, h := range mempoolHashes {
		id, err := hex.DecodeString(h)
		assert.NoError(t, err)
		m.pool[types.NewID(id)] = nil
	}

	blockIDs := make([]types.ID, 0, len(blockHashes))
	for _, h := range blockHashes {
		id, err := hex.DecodeString(h)
		assert.NoError(t, err)
		blockIDs = append(blockIDs, types.NewID(id))
	}

	blk := m.EncodeXthinner(blockIDs)

	//delete(m.pool, blockIDs[1])
	insertID, err := hex.DecodeString("cd1aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29")
	assert.NoError(t, err)
	m.pool[types.NewID(insertID)] = nil

	ret, rerequests := m.DecodeXthinner(blk)

	for _, id := range ret {
		fmt.Println(id.String())
	}
	fmt.Println()
	for _, r := range rerequests {
		fmt.Println(r)
	}

}

func TestBitmapEncoding(t *testing.T) {
	tests := [][]uint32{
		{0, 1, 1, 0, 1, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1},
	}

	for _, test := range tests {
		enc := encodeBitmap(test)
		assert.Equal(t, test, decodeBitmap(enc))
		fmt.Printf("%08b\n", enc)
	}
}
