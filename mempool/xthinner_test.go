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
	tests := []struct {
		blockHashes      []string
		mempoolTxs       []*transactions.Transaction
		mempoolFunc      func(mempool *Mempool)
		expectedTxids    []string
		expectedRequests []uint32
	}{
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // 257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // 257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"0000000000000000000000000000000000000000000000000000000000000000",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			expectedRequests: []uint32{1},
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // 257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"0000000000000000000000000000000000000000000000000000000000000000",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			expectedRequests: []uint32{0},
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // 257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: func(m *Mempool) {
				id, _ := types.NewIDFromString("cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b")
				delete(m.pool, id)
			},
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // 257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 58838}) // cd11927462cd7e99c7bbbda269e9736d545c7e1c8280eace536d4cd815eb9b73
				m.pool[tx.ID()] = tx
			},
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 38973}), // 257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: func(m *Mempool) {
				tx := transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 20870716}) // cd0a76057b7b69b3aca629173f6c7968ed908b843fcb15331def075807180d15
				m.pool[tx.ID()] = tx
			},
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"257973ea2403c0d1080fc5f4ceae5cc08d551c139f19b2ca7215cb4e424519b3",
				"0000000000000000000000000000000000000000000000000000000000000000",
			},
			expectedRequests: []uint32{2},
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"252f180b8a5d799f26a62f6abfdd6013e611c1807db1b67782628b1ddebfeacb",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),   // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}), // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 62961}), // 252f180b8a5d799f26a62f6abfdd6013e611c1807db1b67782628b1ddebfeacb
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}), // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}), // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"252f180b8a5d799f26a62f6abfdd6013e611c1807db1b67782628b1ddebfeacb",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			expectedRequests: nil,
		},
		{
			blockHashes: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"252f03c3441ecb5bf4c57464b2700b7838f69900b7595512ddb340292efb6a79",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			mempoolTxs: []*transactions.Transaction{
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 571}),     // 18f39ef4fbda090e6bb5b32e466bd615a7e1f923d91fefe36141829def2a45fc
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 19200}),   // 252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 8545591}), // 252f03c3441ecb5bf4c57464b2700b7838f69900b7595512ddb340292efb6a79
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 83577}),   // aaa9f54ea56e2abf1874ca594d262c5e23350f10d3cceaf14c95794b233bedcd
				transactions.WrapTransaction(&transactions.StandardTransaction{Fee: 51245}),   // cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b
			},
			mempoolFunc: nil,
			expectedTxids: []string{
				"252f033c1d3a74ca98dc58c4458f9323c1ce1a2b4a4738a84b823426f2a786e5",
				"252f03c3441ecb5bf4c57464b2700b7838f69900b7595512ddb340292efb6a79",
				"cd0a76f607241837aaa2a2bfae056e03671326e292e56ae6094d02e49eeb351b",
			},
			expectedRequests: nil,
		},
	}

	for _, test := range tests {
		m := &Mempool{
			pool: make(map[types.ID]*transactions.Transaction),
		}
		for _, tx := range test.mempoolTxs {
			m.pool[tx.ID()] = tx
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

		for i := range ret.Transactions {
			if test.expectedTxids[i] == "0000000000000000000000000000000000000000000000000000000000000000" {
				assert.Nil(t, ret.Transactions[i])
			} else {
				assert.Equal(t, test.expectedTxids[i], ret.Transactions[i].ID().String())
			}
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

func TestMempool_Close(t *testing.T) {
	i := uint64(51246)
	for {
		tx := transactions.WrapTransaction(&transactions.StandardTransaction{
			Fee: i,
		})
		id := tx.ID()
		if id[0] == 0x25 && id[1] == 0x2f && id[2] == 0x03 {
			fmt.Println(i, id.String())
			break
		}
		i++
	}
}
