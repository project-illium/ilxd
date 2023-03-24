// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMempool(t *testing.T) {
	view := newMockBlockchainView()
	options := []Option{
		DefaultOptions(),
		BlockchainView(view),
	}
	m, err := NewMempool(options...)
	assert.NoError(t, err)
	defer m.Close()

	n := randomID()
	nullifier := types.NewNullifier(n[:])

	txoRoot := randomID()
	view.txoRoots[txoRoot] = true

	tests := []struct {
		name        string
		tx          *transactions.Transaction
		expectedErr error
	}{
		{
			name: "valid standard tx",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier[:]},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        20000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: nil,
		},
		{
			name: "tx already in pool",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier[:]},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        20000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: ErrDuplicateTx,
		},
		{
			name: "standard tx fee too low",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier[:]},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        10000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: policyError(ErrFeeTooLow, "transaction fee is below policy minimum"),
		},
	}

	for _, test := range tests {
		err := m.ProcessTransaction(test.tx)
		if test.expectedErr == nil {
			assert.NoErrorf(t, err, "mempool test: %s failure", test.name)
		} else if _, ok := test.expectedErr.(PolicyError); ok {
			assert.Equalf(t, test.expectedErr.(PolicyError).ErrorCode, err.(PolicyError).ErrorCode, "mempool test: %s: error %s", test.name, err.Error())
		} else if _, ok := test.expectedErr.(blockchain.RuleError); ok {
			assert.Equalf(t, test.expectedErr.(blockchain.RuleError).ErrorCode, err.(blockchain.RuleError).ErrorCode, "mempool test: %s: error %s", test.name, err.Error())
		} else {
			assert.ErrorIsf(t, err, test.expectedErr, "mempool test: %s failure", test.name)
		}
	}
}

func newMockBlockchainView() *mockBlockchainView {
	return &mockBlockchainView{
		treasuryBalance: 0,
		txoRoots:        make(map[types.ID]bool),
		nullifiers:      make(map[types.Nullifier]bool),
		validators:      make(map[peer.ID]types.Amount),
	}
}

func randomID() types.ID {
	r := make([]byte, 32)
	rand.Read(r)
	return types.NewID(r)
}

type mockBlockchainView struct {
	treasuryBalance types.Amount
	txoRoots        map[types.ID]bool
	nullifiers      map[types.Nullifier]bool
	validators      map[peer.ID]types.Amount
}

func (m *mockBlockchainView) TreasuryBalance() (types.Amount, error) {
	return m.treasuryBalance, nil
}

func (m *mockBlockchainView) TxoRootExists(txoRoot types.ID) (bool, error) {
	return m.txoRoots[txoRoot], nil
}

func (m *mockBlockchainView) NullifierExists(n types.Nullifier) (bool, error) {
	return m.nullifiers[n], nil
}

func (m *mockBlockchainView) UnclaimedCoins(validatorID peer.ID) (types.Amount, error) {
	return m.validators[validatorID], nil
}
