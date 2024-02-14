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
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"0000000000000000000000000000000000000000000000000000000000000000",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			expectedRequests: []uint32{1},
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"0000000000000000000000000000000000000000000000000000000000000000",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			expectedRequests: []uint32{0},
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7542}) // 19d5710a9ed7e7832bcdd59c19c4863646e8cd19d6af19b29b307ffd77637ee2
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 1665222}) // 19d58de5cc6e9aab56def8cf9ce7bacd53210e59c1d2b3d599ada53cf9be91e8
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"1798f7616aa6e7de973973d3ac9ff5ec53a3d7b6220c342aa388bb531ce345fe",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),       // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}),    // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2064110}), // 1798f7616aa6e7de973973d3ac9ff5ec53a3d7b6220c342aa388bb531ce345fe
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),      // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),      // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"1798f7616aa6e7de973973d3ac9ff5ec53a3d7b6220c342aa388bb531ce345fe",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"1798f6998590f81399cc26c79d9b3b4a4aac42543933313165886a7ae7259f0f",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),       // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}),    // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2878003}), // 1798f6998590f81399cc26c79d9b3b4a4aac42543933313165886a7ae7259f0f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),      // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),      // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
				"1798f6998590f81399cc26c79d9b3b4a4aac42543933313165886a7ae7259f0f",
				"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
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
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 2}),    // 0bf4c9d8b09c0181a7511b9a7bb8ee416aeef26acd282947a85b51f7663c05cd
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 7127}), // 1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3222}), // 17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 63}),   // 18ee83876b26871be5898d181313b9fc5553ac291210c0ddbc56c45871a5f42a
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 18}),   // 19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29
	}
	for _, tx := range mempoolTxs {
		m.pool[tx.ID()] = &ttlTx{tx: tx}
	}

	blockHashes := []string{
		"1798f664c8349445911f9d1d8a7655e388e47bc6bf022c4d5faa03b384b3ba18",
		"17ebef3b753997dd66cb9f1f2cb25fb11c6fa5dcac65d3ea3d902c6cb863c06c",
		"19d58d02122a4399fc6a25ce3713559408012a47480ee1ba9f9df530845fcf29",
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
