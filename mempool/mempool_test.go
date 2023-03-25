// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package mempool

import (
	"bytes"
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
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

	randomBytes := func() []byte {
		b := make([]byte, 32)
		rand.Read(b)
		return b
	}

	nullifier1 := randomBytes()

	nullifier2 := types.NewNullifier(randomBytes())
	view.nullifiers[nullifier2] = true

	view.treasuryBalance = 30000

	txoRoot := randomID()
	txoRoot2 := randomID()
	view.txoRoots[txoRoot] = true

	sk, pk, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	pkBytes, err := crypto.MarshalPublicKey(pk)
	assert.NoError(t, err)
	validatorID, err := peer.IDFromPublicKey(pk)
	assert.NoError(t, err)
	valBytes, err := validatorID.Marshal()
	assert.NoError(t, err)

	view.validators[validatorID] = 10000

	tests := []struct {
		name        string
		tx          *transactions.Transaction
		signFunc    func(tx *transactions.Transaction) error
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
				Nullifiers: [][]byte{nullifier1},
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
				Nullifiers: [][]byte{nullifier1[:]},
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
				Nullifiers: [][]byte{randomBytes()},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        10000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: policyError(ErrFeeTooLow, "transaction fee is below policy minimum"),
		},
		{
			name: "standard nullifier already in pool",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier1},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        30000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: ruleError(blockchain.ErrDoubleSpend, ""),
		},
		{
			name: "standard nullifier already in set",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier2[:]},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        30000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: ruleError(blockchain.ErrDoubleSpend, ""),
		},
		{
			name: "standard txo root not in set",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{randomBytes()},
				TxoRoot:    txoRoot2[:],
				Locktime:   0,
				Fee:        30000,
				Proof:      make([]byte, 1000),
			}),
			expectedErr: ruleError(blockchain.ErrInvalidTx, ""),
		},
		{
			name: "valid mint tx",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Asset_ID: pkBytes,
				Type:     transactions.MintTransaction_VARIABLE_SUPPLY,
				MintKey:  pkBytes,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{randomBytes()},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        20000,
				Proof:      make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetMintTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetMintTransaction().Signature = sig
				return nil
			},
			expectedErr: nil,
		},
		{
			name: "mint tx fee too low",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Asset_ID: pkBytes,
				Type:     transactions.MintTransaction_VARIABLE_SUPPLY,
				MintKey:  pkBytes,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{randomBytes()},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        10000,
				Proof:      make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetMintTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetMintTransaction().Signature = sig
				return nil
			},
			expectedErr: policyError(ErrFeeTooLow, "transaction fee is below policy minimum"),
		},
		{
			name: "mint nullifier already in pool",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Asset_ID: pkBytes,
				Type:     transactions.MintTransaction_VARIABLE_SUPPLY,
				MintKey:  pkBytes,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier1[:]},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        30000,
				Proof:      make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetMintTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetMintTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrDoubleSpend, ""),
		},
		{
			name: "mint nullifier already in set",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Asset_ID: pkBytes,
				Type:     transactions.MintTransaction_VARIABLE_SUPPLY,
				MintKey:  pkBytes,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier2[:]},
				TxoRoot:    txoRoot[:],
				Locktime:   0,
				Fee:        30000,
				Proof:      make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetMintTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetMintTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrDoubleSpend, ""),
		},
		{
			name: "mint txo root not in set",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Asset_ID: pkBytes,
				Type:     transactions.MintTransaction_VARIABLE_SUPPLY,
				MintKey:  pkBytes,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Nullifiers: [][]byte{randomBytes()},
				TxoRoot:    txoRoot2[:],
				Locktime:   0,
				Fee:        30000,
				Proof:      make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetMintTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetMintTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrInvalidTx, ""),
		},
		{
			name: "coinbase tx invalid coins",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: valBytes,
				NewCoins:     20000,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Signature: nil,
				Proof:     make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetCoinbaseTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetCoinbaseTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrInvalidTx, ""),
		},
		{
			name: "valid coinbase tx",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: valBytes,
				NewCoins:     10000,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Signature: nil,
				Proof:     make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetCoinbaseTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetCoinbaseTransaction().Signature = sig
				return nil
			},
			expectedErr: nil,
		},
		{
			name: "coinbase from validator already exsits",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: valBytes,
				NewCoins:     10000,
				Outputs: []*transactions.Output{
					{
						Commitment:      bytes.Repeat([]byte{0x11}, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Signature: nil,
				Proof:     make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetCoinbaseTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetCoinbaseTransaction().Signature = sig
				return nil
			},
			expectedErr: policyError(ErrDuplicateCoinbase, ""),
		},
		{
			name: "valid coinbase replacement",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: valBytes,
				NewCoins:     20000,
				Outputs: []*transactions.Output{
					{
						Commitment:      bytes.Repeat([]byte{0x11}, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
				Signature: nil,
				Proof:     make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				view.validators[validatorID] = 20000
				h, err := tx.GetCoinbaseTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetCoinbaseTransaction().Signature = sig
				return nil
			},
			expectedErr: nil,
		},
		{
			name: "stake below minimum",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Validator_ID: valBytes,
				Amount:       m.cfg.minStake - 1,
				Nullifier:    randomBytes(),
				TxoRoot:      txoRoot[:],
				Proof:        make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetStakeTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetStakeTransaction().Signature = sig
				return nil
			},
			expectedErr: policyError(ErrMinStake, ""),
		},
		{
			name: "valid stake transaction",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Validator_ID: valBytes,
				Amount:       m.cfg.minStake,
				Nullifier:    randomBytes(),
				TxoRoot:      txoRoot[:],
				Proof:        make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetStakeTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetStakeTransaction().Signature = sig
				return nil
			},
			expectedErr: nil,
		},
		{
			name: "stake nullifier already in pool",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Validator_ID: valBytes,
				Amount:       m.cfg.minStake,
				Nullifier:    nullifier1,
				TxoRoot:      txoRoot[:],
				Proof:        make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetStakeTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetStakeTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrDoubleSpend, ""),
		},
		{
			name: "stake nullifier already in set",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Validator_ID: valBytes,
				Amount:       m.cfg.minStake,
				Nullifier:    nullifier2[:],
				TxoRoot:      txoRoot[:],
				Proof:        make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetStakeTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetStakeTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrDoubleSpend, ""),
		},
		{
			name: "stake txo root not in set",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Validator_ID: valBytes,
				Amount:       m.cfg.minStake,
				Nullifier:    randomBytes(),
				TxoRoot:      txoRoot2[:],
				Proof:        make([]byte, 1000),
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h, err := tx.GetStakeTransaction().SigHash()
				if err != nil {
					return err
				}
				sig, err := sk.Sign(h)
				if err != nil {
					return err
				}
				tx.GetStakeTransaction().Signature = sig
				return nil
			},
			expectedErr: ruleError(blockchain.ErrInvalidTx, ""),
		},
		{
			name: "treasury over limit",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				Amount: 40000,
				Outputs: []*transactions.Output{
					{
						Commitment:      bytes.Repeat([]byte{0x11}, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h := tx.ID()
				m.cfg.treasuryWhitelist[h] = true
				return nil
			},
			expectedErr: ruleError(blockchain.ErrInvalidTx, ""),
		},
		{
			name: "valid treasury tx",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				Amount: 20000,
				Outputs: []*transactions.Output{
					{
						Commitment:      bytes.Repeat([]byte{0x11}, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h := tx.ID()
				m.cfg.treasuryWhitelist[h] = true
				return nil
			},
			expectedErr: nil,
		},
		{
			name: "treasury + pool over limit",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				Amount: 10001,
				Outputs: []*transactions.Output{
					{
						Commitment:      bytes.Repeat([]byte{0x11}, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
			}),
			signFunc: func(tx *transactions.Transaction) error {
				h := tx.ID()
				m.cfg.treasuryWhitelist[h] = true
				return nil
			},
			expectedErr: ruleError(blockchain.ErrInvalidTx, ""),
		},
		{
			name: "treasury not whitelisted",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				Amount: 10000,
				Outputs: []*transactions.Output{
					{
						Commitment:      bytes.Repeat([]byte{0x11}, types.CommitmentLen),
						EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
						Ciphertext:      make([]byte, blockchain.CiphertextLen),
					},
				},
			}),
			expectedErr: policyError(ErrTreasuryWhitelist, ""),
		},
	}

	for _, test := range tests {
		if test.signFunc != nil {
			err := test.signFunc(test.tx)
			assert.NoError(t, err)
		}
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

func TestFeePerByte(t *testing.T) {
	tx := transactions.WrapTransaction(&transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment:      make([]byte, types.CommitmentLen),
				EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
				Ciphertext:      make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{make([]byte, 32)},
		TxoRoot:    make([]byte, 32),
		Locktime:   0,
		Fee:        20000,
		Proof:      make([]byte, 1000),
	})
	size, err := tx.SerializedSize()
	assert.NoError(t, err)

	fpb, ok, err := calcFeePerByte(tx)
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetStandardTransaction().Fee/uint64(size), fpb)
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
