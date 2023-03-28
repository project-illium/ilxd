// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"encoding/hex"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMempool_EncodeXthinner(t *testing.T) {
	tests := []struct {
		blockHashes      []string
		mempoolHashes    []string
		mempoolFunc      func(mempool *Mempool)
		expectedTxids    []string
		expectedRequests []uint32
	}{
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"0000000000000000000000000000000000000000000000000000000000000000",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			expectedRequests: []uint32{1},
		},
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"0000000000000000000000000000000000000000000000000000000000000000",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			expectedRequests: []uint32{0},
		},
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("cd11a9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29")
				m.pool[id] = nil
			},
			expectedTxids: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("cd0aa9856147b6c5b4ffbc52c9993eb66aeb366602958fdfaa283b71e64db123")
				m.pool[id] = nil
			},
			expectedTxids: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"2579bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"252fbb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"252fbb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"252fbb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"252fbb7b435b053216519c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"252fbb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolHashes: []string{
				"18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4",
				"252fbb7b435b053216519c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"252fbb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"aaa9402664f1a41f40ebbc52c9993eb66aeb366602958fdfaa283b71e64db123",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"252fbb7b435b053216519c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
				"252fbb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea",
				"cd0aa9856147b6c5b4ff2b7dfee5da20aa38253099ef1b4a64aced233c9afe29",
			},
			expectedRequests: nil,
		},
	}

	for _, test := range tests {
		m := &Mempool{
			pool: make(map[types.ID]*transactions.Transaction),
		}
		for _, h := range test.mempoolHashes {
			id, err := hex.DecodeString(h)
			assert.NoError(t, err)
			m.pool[types.NewID(id)] = nil
		}

		blockIDs := make([]types.ID, 0, len(test.blockHashes))
		for _, h := range test.blockHashes {
			id, err := hex.DecodeString(h)
			assert.NoError(t, err)
			blockIDs = append(blockIDs, types.NewID(id))
		}

		blk, err := m.EncodeXthinner(blockIDs)
		assert.NoError(t, err)

		if test.mempoolFunc != nil {
			test.mempoolFunc(m)
		}

		ret, rerequests := m.DecodeXthinner(blk)

		for i := range ret {
			assert.Equal(t, test.expectedTxids[i], ret[i].String())
		}

		for i := range rerequests {
			assert.Equal(t, test.expectedRequests[i], rerequests[i])
		}
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
		if len(test)%8 != 0 {
			test = append(test, make([]uint32, 8-len(test)%8)...)
		}
		assert.Equal(t, test, decodeBitmap(enc))
	}
}
