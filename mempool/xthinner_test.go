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
		mempoolTxs       []*transactions.Transaction
		mempoolFunc      func(mempool *Mempool)
		expectedTxids    []string
		expectedRequests []uint32
	}{
		{
			blockHashes: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 313}),   // a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 313}),   // a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"0000000000000000000000000000000000000000000000000000000000000000",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			expectedRequests: []uint32{1},
		},
		{
			blockHashes: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 313}),   // a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"0000000000000000000000000000000000000000000000000000000000000000",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			expectedRequests: []uint32{0},
		},
		{
			blockHashes: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 313}),   // a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 313}),   // a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 207}) // f8768e8090f09645358085c23439796579d380ef8b430f7085fd7dcf89c2dfcd
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 313}),   // a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18552267}) // f876d5a735409ee6d2864f0ece7199011c13746151ce449bb089bfd4228a6152
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"a7eb22ce649595789acb1a40b5be22409554edd998843ae95c7b74b4dc54560f",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"a798055057b39b7f26ebe0e72fc2768f60ca41b5ed66b7ae6b6858d07a6b84db",
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 97286}), // a798055057b39b7f26ebe0e72fc2768f60ca41b5ed66b7ae6b6858d07a6b84db
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"a798055057b39b7f26ebe0e72fc2768f60ca41b5ed66b7ae6b6858d07a6b84db",
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"a798f493d0b05c9fd373dddbd6236a39354bbbfd6b7452dbd71bb0d31299553f",
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}),    // 0bc4757779ffb01681696ff4fe8ef43dff4411f81a7c173f55b1601f54fe9097
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 25003210}), // a798f493d0b05c9fd373dddbd6236a39354bbbfd6b7452dbd71bb0d31299553f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}),    // a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}),    // e11870bd0ca0cd4c846ec100ca264f9968866c4381bcf6f0750b8a89d5b97d84
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),      // f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"a798f493d0b05c9fd373dddbd6236a39354bbbfd6b7452dbd71bb0d31299553f",
				"a798f4c1534852561d7273a2b8331241f5f8caaad4c31d5b43f07db627f6ea5a",
				"f876d51d4b0b73ffa0bd5721545cf0365c59289b4267dc84311af9abc6488f7d",
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
