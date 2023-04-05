// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"bytes"
	"errors"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"math/rand"
	"sort"
	"time"
)

// EncodeXthinner replaces the full transactions from a block.Block with
// short transaction prefixes and returns an XThinnerBlock.  The idea being
// that the receiving node can use the prefixes and their own mempool to
// reconstruct the full block. Because block transactions are sorted by
// txid by consensus rule, we can send just the minimum number of bytes needed
// to disambiguate between transactions in our mempool.
//
// The Xthinner spec calls for the XthinnerBlock to have a checksum attached
// that can be used to detect collisions. However, in illium blocks come
// quickly and typically have few transactions. With such a small number of
// transactions collisions are less likely. In the event a collision happens
// the validation of the txRoot will fail and we will just request the full
// list of block txids from the relaying peer.
//
// If checksums were included we could use them to narrow down the range in
// block where the collision occurred and just request a smaller range of
// txids from our peer instead of the full list. This would save bandwidth.
// However, for it to be worth it the bandwidth savings needs to exceed
// the extra bandwidth cost of the checksums. At small block sizes this isn't
// worth it. So we omit the checksums.
func (m *Mempool) EncodeXthinner(blkIds []types.ID) (*blocks.XThinnerBlock, error) {
	m.mempoolLock.RLock()
	defer m.mempoolLock.RUnlock()

	rand.Seed(time.Now().Unix())

	mempoolTxs := make([]types.ID, 0, len(m.pool)+2)

	for txid := range m.pool {
		mempoolTxs = append(mempoolTxs, txid)
	}
	sort.Sort(TxidSorter(mempoolTxs))
	mempoolTxs = append(mempoolTxs, types.NewID(make([]byte, 32)))

	var (
		stack      = make([]byte, 0, 32)
		pops       = make([]uint32, 0, 32)
		pushes     = make([]uint32, 0, 32)
		pushbytes  = make([]byte, 0, 32)
		mempoolPos int
	)

	for _, txid := range blkIds {
		if _, ok := m.pool[txid]; !ok {
			return nil, errors.New("block tx not found in pool")
		}
		// Pop stage
		if len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
		for i := 0; i < len(stack); i++ {
			if stack[i] != txid[i] {
				l := len(stack) - i
				for j := 0; j < l; j++ {
					stack = stack[:len(stack)-1]
					pops = append(pops, 1)
				}
				break
			}
		}
		pops = append(pops, 0)

		// Push stage
		for mempoolTxs[mempoolPos] != txid {
			if mempoolPos+1 == len(mempoolTxs) {
				mempoolPos = 0
			} else {
				mempoolPos++
			}
		}
		stack = append(stack, txid[len(stack)])
		pushbytes = append(pushbytes, stack[len(stack)-1])

		r := mempoolPos
		if r-1 < 0 {
			r = len(mempoolTxs) - 1
		}
		prev := mempoolTxs[r-1]
		next := mempoolTxs[mempoolPos+1]

		for (bytes.Equal(stack, prev[:len(stack)]) || bytes.Equal(stack, next[:len(stack)])) && len(mempoolTxs) > 2 {
			pushes = append(pushes, 1)
			stack = append(stack, txid[len(stack)])
			pushbytes = append(pushbytes, stack[len(stack)-1])
		}
		pushes = append(pushes, 0)

		mempoolPos++
	}

	return &blocks.XThinnerBlock{
		Header:       nil,
		TxCount:      uint32(len(blkIds)),
		Pops:         encodeBitmap(pops),
		Pushes:       encodeBitmap(pushes),
		PushBytes:    pushbytes,
		PrefilledTxs: nil,
	}, nil
}

// DecodeXthinner decodes an XThinnerBlock using the transactions in the mempool.
// There are two possible decode failures:
//  1. There are no transactions in the mempool with the given prefix (missing tx).
//  2. There are one or more transactions in the mempool with the same prefix (collision).
//
// In both scenarios nil is returned for that transaction and the missing index is
// appended to the returned []uint32
//
// The caller should request the missing transaction(s) from the remote peer before proceeding
// to block validation.
//
// The final failure case is the block's tx merkle root may fail to validate. This could
// mean one of two things:
//  1. We had a transaction in the mempool that collided with a transaction in the block,
//     but the block's transaction was not in the mempool. This is very unlikely but
//     possible.
//  2. The peer maliciously sent us an invalid block.
//
// In both cases the solution is to download the full list of txids from the remote peer.
func (m *Mempool) DecodeXthinner(blk *blocks.XThinnerBlock) (*blocks.Block, []uint32) {
	m.mempoolLock.RLock()
	defer m.mempoolLock.RUnlock()

	var (
		pops            = decodeBitmap(blk.Pops)
		pushes          = decodeBitmap(blk.Pushes)
		popi            = 0
		pushi           = 0
		pushbi          = 0
		mempoolposition = 0
		stack           = make([]byte, 0, 32)
		fullBlk         = &blocks.Block{Header: blk.Header, Transactions: make([]*transactions.Transaction, blk.TxCount)}
		rerequests      = make([]uint32, 0, blk.TxCount)
	)

	mempoolTxs := make([]types.ID, 0, len(m.pool)+1)

	for txid := range m.pool {
		mempoolTxs = append(mempoolTxs, txid)
	}
	prefilled := make(map[types.ID]*transactions.Transaction)
	for _, tx := range blk.PrefilledTxs {
		id := tx.Transaction.ID()
		if _, ok := m.pool[id]; !ok {
			mempoolTxs = append(mempoolTxs, id)
		}
		prefilled[id] = tx.Transaction
	}
	sort.Sort(TxidSorter(mempoolTxs))
	mempoolTxs = append(mempoolTxs, types.NewID(bytes.Repeat([]byte{0xff}, 32)))

	for pos := uint32(0); pos < blk.TxCount; pos++ {
		if len(stack) > 0 {
			stack = stack[:len(stack)-1] // 0x25
		}
		pop := pops[popi]
		popi++

		for pop > 0 {
			stack = stack[:len(stack)-1]
			pop = pops[popi]
			popi++
		}

		stack = append(stack, blk.PushBytes[pushbi])
		pushbi++
		push := pushes[pushi]
		pushi++

		for push > 0 {
			stack = append(stack, blk.PushBytes[pushbi])
			pushbi++
			push = pushes[pushi]
			pushi++
		}

		for mempoolposition < len(mempoolTxs)-1 && types.NewID(stack).Compare(types.NewID(mempoolTxs[mempoolposition][:len(stack)])) > 0 {
			mempoolposition++
		}
		// Transaction not found in mempool
		if types.NewID(stack).Compare(types.NewID(mempoolTxs[mempoolposition][:len(stack)])) != 0 {
			fullBlk.Transactions[pos] = nil
			rerequests = append(rerequests, pos)
			// More than one transaction in the mempool with the same prefix. Unable to disambiguate.
		} else if mempoolposition < len(mempoolTxs)-2 && bytes.Equal(mempoolTxs[mempoolposition+1][:len(stack)], stack) {
			fullBlk.Transactions[pos] = nil
			rerequests = append(rerequests, pos)
		} else if tx, ok := prefilled[mempoolTxs[mempoolposition]]; ok {
			fullBlk.Transactions[pos] = tx
		} else {
			fullBlk.Transactions[pos] = m.pool[mempoolTxs[mempoolposition]]
		}
	}
	return fullBlk, rerequests
}

func encodeBitmap(bits []uint32) []byte {
	if len(bits)%8 != 0 {
		bits = append(bits, make([]uint32, 8-len(bits)%8)...)
	}
	ret := make([]byte, (len(bits)+7)/8)
	for i := range ret {
		for x := 0; x < 8; x++ {
			ret[i] <<= 1
			if bits[x+(8*i)] > 0 {
				ret[i] |= 0x01
			}
		}
	}
	return ret
}

func decodeBitmap(bm []byte) []uint32 {
	bits := make([]uint32, len(bm)*8)
	for i, b := range bm {
		for x := 0; x < 8; x++ {
			y := byte(0x01 << (7 - x))
			if b&y > 0 {
				bits[x+(8*i)] = 1
			}

		}
	}
	return bits
}
