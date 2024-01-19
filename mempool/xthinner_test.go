// Copyright (c) 2024 The illium developers
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
		mempoolTxs       []*transactions.Transaction
		mempoolFunc      func(mempool *Mempool)
		expectedTxids    []string
		expectedRequests []uint32
	}{
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"0000000000000000000000000000000000000000000000000000000000000000",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			expectedRequests: []uint32{1},
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"0000000000000000000000000000000000000000000000000000000000000000",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			expectedRequests: []uint32{0},
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}) // 3876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 5561605}) // 38768ed35c7c101c5f44cf8fb753206b616c36f8d5e1256b5a52a9ca76f47493
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"1798cbf7dde043007f112e9e393b218cac9a9d10e07a469a6d8d86c0ccaa3631",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 77507}), // 1798cbf7dde043007f112e9e393b218cac9a9d10e07a469a6d8d86c0ccaa3631
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"1798cbf7dde043007f112e9e393b218cac9a9d10e07a469a6d8d86c0ccaa3631",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"179845d17bfc6c2a212670ae25ac3d1167b914e45b6d3356ec5ccdab6e6f9715",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}),    // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}),    // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 35405036}), // 179845d17bfc6c2a212670ae25ac3d1167b914e45b6d3356ec5ccdab6e6f9715
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),      // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),      // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
				"179845d17bfc6c2a212670ae25ac3d1167b914e45b6d3356ec5ccdab6e6f9715",
				"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
			},
			expectedRequests: nil,
		},
	}

	for x, test := range tests {
		m := &Mempool{
			pool: make(map[types.ID]*ttlTx),
		}
		for _, tx := range test.mempoolTxs {
			m.pool[tx.ID()] = &ttlTx{tx: tx}
		}

		blockIDs := make([]types.ID, 0, len(test.blockHashes))
		for _, h := range test.blockHashes {
			id, err := hex.DecodeString(h)
			assert.NoErrorf(t, err, "test %d", x)
			blockIDs = append(blockIDs, types.NewID(id))
		}

		blk, err := m.EncodeXthinner(blockIDs)
		assert.NoErrorf(t, err, "test %d", x)

		if test.mempoolFunc != nil {
			test.mempoolFunc(m)
		}

		ret, rerequests := m.DecodeXthinner(blk)

		for i := range ret.Transactions {
			if test.expectedTxids[i] == "0000000000000000000000000000000000000000000000000000000000000000" {
				assert.Nilf(t, ret.Transactions[i], "test %d", x)
			} else {
				assert.Equalf(t, test.expectedTxids[i], ret.Transactions[i].ID().String(), "test %d", x)
			}
		}

		for i := range rerequests {
			assert.Equalf(t, test.expectedRequests[i], rerequests[i], "test %d", x)
		}
	}
}
func TestEmptyPool(t *testing.T) {
	m := &Mempool{
		pool: make(map[types.ID]*ttlTx),
	}
	mempoolTxs := []*transactions.Transaction{
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 11656}), // 0bc46d546b12fbef86d09c33b0830e259e73ebec4940f7027e562c67421aa2ab
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20489}), // 179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 12572}), // 17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 803}),   // 211810a143c8a24e85c427935cc7685c93357d9813501e49c4999da6f1e5a891
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}),   // 38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
	}
	for _, tx := range mempoolTxs {
		m.pool[tx.ID()] = &ttlTx{tx: tx}
	}

	blockHashes := []string{
		"179845cb2e2157fe43eff08071aa6f61f41a3060d5b9dd5a633daf1f11735f4a",
		"17eb2a0819a24e397fd29216c2eed73ef6fd9bee5cee0b28328adf8c5cbb526d",
		"38768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd",
	}

	blockIDs := make([]types.ID, 0, len(blockHashes))
	for _, h := range blockHashes {
		id, err := hex.DecodeString(h)
		assert.NoError(t, err)
		blockIDs = append(blockIDs, types.NewID(id))
	}

	blk, err := m.EncodeXthinner(blockIDs)
	assert.NoError(t, err)

	m2 := &Mempool{
		pool: make(map[types.ID]*ttlTx),
	}

	_, missing := m2.DecodeXthinner(blk)
	assert.Len(t, missing, 3)
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
