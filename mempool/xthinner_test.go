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
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"0000000000000000000000000000000000000000000000000000000000000000",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			expectedRequests: []uint32{1},
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"0000000000000000000000000000000000000000000000000000000000000000",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			expectedRequests: []uint32{0},
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4443}) // 381144b95dd2353ea3a63a30c0d255eb39df628e40db4e238224bcff8efc2c84
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 3018673}) // 3811591e7291c60bbe97868043a2514f002c2945f3b82715b86bae0f09e1a86b
				m.pool[tx.ID()] = &ttlTx{tx: tx}
			},
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"179898059c393db7f1fd0e3cd7c7bef5f8029a8ffe3604e63389b09ce839b74f",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),    // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}), // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 6239}), // 179898059c393db7f1fd0e3cd7c7bef5f8029a8ffe3604e63389b09ce839b74f
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),  // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),  // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"179898059c393db7f1fd0e3cd7c7bef5f8029a8ffe3604e63389b09ce839b74f",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17981babbd8eb5119f71f6aafa441d577eb6304d999d7edb69f998ec6bee05b4",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),      // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),   // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 199055}), // 17981babbd8eb5119f71f6aafa441d577eb6304d999d7edb69f998ec6bee05b4
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),    // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),    // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
				"17981babbd8eb5119f71f6aafa441d577eb6304d999d7edb69f998ec6bee05b4",
				"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
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
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 0}),     // 0b0b3de5d4aa28ad571cfb0e139b0a41684ca92dc444acd4705b59aa00a0af79
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 4868}),  // 17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 16593}), // 17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 182}),   // 2151c2068f78e47bd3f3fb42d5d597b455a5cb290072f8fb3a6ba599e2d8cfb9
		transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 104}),   // 381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0
	}
	for _, tx := range mempoolTxs {
		m.pool[tx.ID()] = &ttlTx{tx: tx}
	}

	blockHashes := []string{
		"17981b23d40fd9530c91d1ba5f897ab9d5420f43e1731dcc965543b94df374a5",
		"17eb09407b6b1241c5edc4f34ce5fb9b1236db0c3813701439a90106bb5fb574",
		"381159358e007ed1d00d90a207eef0561260ef7bea9803a4944598580e0276b0",
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
