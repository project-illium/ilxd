// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

func TestValidateHeader(t *testing.T) {
	b := Blockchain{}

	// Build valid header
	validHeader := randomBlockHeader(1, randomID())
	sk, pk, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	pid, err := peer.IDFromPublicKey(pk)
	assert.NoError(t, err)

	pidBytes, err := pid.Marshal()
	assert.NoError(t, err)
	validHeader.Producer_ID = pidBytes

	sigHash, err := validHeader.SigHash()
	assert.NoError(t, err)

	sig, err := sk.Sign(sigHash)
	assert.NoError(t, err)
	validHeader.Signature = sig

	tests := []struct {
		name        string
		header      *blocks.BlockHeader
		flags       BehaviorFlags
		expectedErr error
	}{
		{
			name:        "header with no signature",
			header:      randomBlockHeader(1, randomID()),
			flags:       BFNone,
			expectedErr: ruleError(ErrInvalidHeaderSignature, ""),
		},
		{
			name:        "header with no signature fast add",
			header:      randomBlockHeader(1, randomID()),
			flags:       BFFastAdd,
			expectedErr: nil,
		},
		{
			name:        "genesis validation",
			header:      params.RegestParams.GenesisBlock.Header,
			flags:       BFGenesisValidation,
			expectedErr: nil,
		},
		{
			name:        "header with no producer ID",
			header:      &blocks.BlockHeader{},
			flags:       BFNone,
			expectedErr: ruleError(ErrInvalidProducer, ""),
		},
		{
			name:        "valid header",
			header:      validHeader,
			flags:       BFNone,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		err = b.validateHeader(test.header, test.flags)
		if test.expectedErr == nil {
			assert.NoErrorf(t, err, "header validation test: %s failure", test.name)
		} else {
			assert.Equal(t, test.expectedErr.(RuleError).ErrorCode, err.(RuleError).ErrorCode, "header validation test: %s failure", test.name)
		}
	}
}

func TestCheckBlockContext(t *testing.T) {
	ds := mock.NewMapDatastore()
	err := populateDatabase(ds, 5000)
	assert.NoError(t, err)

	index := NewBlockIndex(ds)
	err = index.Init()
	assert.NoError(t, err)

	vs := NewValidatorSet(&params.RegestParams, ds)
	validatorID := randomPeerID()
	valBytes, err := validatorID.Marshal()
	assert.NoError(t, err)
	validatorID2 := randomPeerID()
	valBytes2, err := validatorID2.Marshal()
	assert.NoError(t, err)
	vs.validators[validatorID] = &Validator{
		PeerID: validatorID,
	}

	b := Blockchain{
		index:        index,
		validatorSet: vs,
	}

	prev, err := index.Tip().Header()
	assert.NoError(t, err)
	prevID := prev.ID()

	tests := []struct {
		name        string
		header      *blocks.BlockHeader
		expectedErr error
	}{
		{
			name: "valid header",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 1,
				Parent:      prevID[:],
				Timestamp:   prev.Timestamp + 1,
				Producer_ID: valBytes,
			},
			expectedErr: nil,
		},
		{
			name: "height doesn't connect",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 2,
				Parent:      prevID[:],
				Timestamp:   prev.Timestamp + 1,
				Producer_ID: valBytes,
			},
			expectedErr: ruleError(ErrDoesNotConnect, ""),
		},
		{
			name: "paraent doesn't connect",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 1,
				Parent:      []byte{0x00},
				Timestamp:   prev.Timestamp + 1,
				Producer_ID: valBytes,
			},
			expectedErr: ruleError(ErrDoesNotConnect, ""),
		},
		{
			name: "timestamp too early",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 1,
				Parent:      prevID[:],
				Timestamp:   prev.Timestamp,
				Producer_ID: valBytes,
			},
			expectedErr: ruleError(ErrInvalidTimestamp, ""),
		},
		{
			name: "invalid producer ID",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 1,
				Parent:      prevID[:],
				Timestamp:   prev.Timestamp + 1,
				Producer_ID: []byte{0x00},
			},
			expectedErr: ruleError(ErrInvalidProducer, ""),
		},
		{
			name: "producer not in set",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 1,
				Parent:      prevID[:],
				Timestamp:   prev.Timestamp + 1,
				Producer_ID: valBytes2,
			},
			expectedErr: ruleError(ErrInvalidProducer, ""),
		},
	}

	for _, test := range tests {
		err = b.checkBlockContext(test.header)
		if test.expectedErr == nil {
			assert.NoErrorf(t, err, "block context test: %s failure", test.name)
		} else {
			assert.Equal(t, test.expectedErr.(RuleError).ErrorCode, err.(RuleError).ErrorCode, "block context test: %s failure", test.name)
		}
	}
}

func TestValidateBlock(t *testing.T) {
	ds := mock.NewMapDatastore()
	b := Blockchain{
		txoRootSet:   NewTxoRootSet(ds, 10),
		nullifierSet: NewNullifierSet(ds, 10),
	}

	// Build valid block
	header := randomBlockHeader(1, randomID())
	sk, pk, err := crypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)

	pid, err := peer.IDFromPublicKey(pk)
	assert.NoError(t, err)

	pidBytes, err := pid.Marshal()
	assert.NoError(t, err)
	header.Producer_ID = pidBytes

	block := &blocks.Block{
		Header:       header,
		Transactions: nil,
	}

	var (
		txoRoot   = randomID()
		nullifier = randomID()
	)
	b.txoRootSet.cache[txoRoot] = true

	signHeader := func(header *blocks.BlockHeader) (*blocks.BlockHeader, error) {
		sigHash, err := header.SigHash()
		if err != nil {
			return nil, err
		}

		sig, err := sk.Sign(sigHash)
		if err != nil {
			return nil, err
		}
		header.Signature = sig
		return header, nil
	}

	tests := []struct {
		name        string
		block       func(blk *blocks.Block) (*blocks.Block, error)
		flags       BehaviorFlags
		expectedErr error
	}{
		{
			name: "block has no transactions",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrEmptyBlock, ""),
		},
		{
			name: "invalid transaction root",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{}),
				}
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrInvalidTxRoot, ""),
		},
		{
			name: "standard transaction during genesis validation",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment:      make([]byte, types.CommitmentLen),
								EphemeralPubkey: make([]byte, PubkeyLen),
								Ciphertext:      make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkles := BuildMerkleTreeStore(blk.Transactions)
				header.TxRoot = merkles[len(merkles)-1]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFGenesisValidation,
			expectedErr: ruleError(ErrInvalidGenesis, ""),
		},
		{
			name: "standard transaction no nullifiers",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment:      make([]byte, types.CommitmentLen),
								EphemeralPubkey: make([]byte, PubkeyLen),
								Ciphertext:      make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkles := BuildMerkleTreeStore(blk.Transactions)
				header.TxRoot = merkles[len(merkles)-1]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
	}

	for _, test := range tests {
		blk, err := test.block(proto.Clone(block).(*blocks.Block))
		assert.NoError(t, err)
		err = b.validateBlock(blk, test.flags)
		if test.expectedErr == nil {
			assert.NoErrorf(t, err, "block validation test: %s failure", test.name)
		} else {
			assert.Equalf(t, test.expectedErr.(RuleError).ErrorCode, err.(RuleError).ErrorCode, "block validation test: %s: error %s", test.name, err.Error())
		}
	}
}

func TestCheckTransactionSanity(t *testing.T) {
	peerID := randomPeerID()
	peerIDBytes, err := peerID.Marshal()
	assert.NoError(t, err)

	n := randomID()
	nullifier := types.NewNullifier(n[:])

	tests := []struct {
		name        string
		tx          *transactions.Transaction
		timestamp   time.Time
		expectedErr error
	}{
		{
			name:        "missing tx",
			tx:          transactions.WrapTransaction(nil),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase valid",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: peerIDBytes,
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "coinbase no outputs",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: peerIDBytes,
				Outputs:      []*transactions.Output{},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase invalid validator ID",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: []byte{},
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase invalid commitment len",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: []byte{},
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen+1),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase invalid pubkey len",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: []byte{},
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen+1),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase invalid ciphertext len",
			tx: transactions.WrapTransaction(&transactions.CoinbaseTransaction{
				Validator_ID: []byte{},
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen+1),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard valid",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "standard no outputs",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Outputs:    []*transactions.Output{},
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard no nullifiers",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{},
				Locktime:   time.Time{}.Unix(),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard invalid commitment len",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen+1),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard invalid pubkey len",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen+1),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard invalid ciphertext len",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen+1),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard invalid locktime",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Now().Add(time.Hour).Unix(),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "stake valid",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nullifier.Bytes(),
				Validator_ID: peerIDBytes,
				Locktime:     time.Time{}.Unix(),
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "stake invalid locktime",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nullifier.Bytes(),
				Validator_ID: peerIDBytes,
				Locktime:     time.Now().Add(time.Hour).Unix(),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "stake invalid validator ID",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nullifier.Bytes(),
				Validator_ID: nil,
				Locktime:     time.Time{}.Unix(),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "stake invalid nullifier",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nil,
				Validator_ID: peerIDBytes,
				Locktime:     time.Time{}.Unix(),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint valid fixed supply",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				Locktime:   time.Time{}.Unix(),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "mint valid variable supply",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Type:     transactions.MintTransaction_VARIABLE_SUPPLY,
				Asset_ID: bytes.Repeat([]byte{0x11}, 32),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier.Bytes()},
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Locktime:   time.Time{}.Unix(),
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "mint no outputs",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Outputs:    []*transactions.Output{},
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint no nullifiers",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid commitment len",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen+1),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid pubkey len",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen+1),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid ciphertext len",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen+1),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid locktime",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Now().Add(time.Hour).Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid asset ID len",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   []byte{0x01},
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid mint key len",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    []byte{0x01},
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid fixed supply asset ID",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   bytes.Repeat([]byte{0x11}, AssetIDLen),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid variable supply invalid asset ID",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Type:       transactions.MintTransaction_VARIABLE_SUPPLY,
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   types.IlliumCoinID[:],
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid variable supply assetID doesn't match key",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Type:       transactions.MintTransaction_VARIABLE_SUPPLY,
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   bytes.Repeat([]byte{0x22}, AssetIDLen),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint unknown type",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Type:       3,
				Locktime:   time.Time{}.Unix(),
				Asset_ID:   bytes.Repeat([]byte{0x11}, AssetIDLen),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrUnknownTxEnum, ""),
		},
		{
			name: "treasury valid",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxProposalHashLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "treasury no outputs",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				Outputs: []*transactions.Output{},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "treasury invalid proposal hash",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxProposalHashLen+1),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "treasury invalid commitment len",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxProposalHashLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen+1),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "treasury invalid pubkey len",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxProposalHashLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen+1),
						Ciphertext:      make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "treasury invalid ciphertext len",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxProposalHashLen),
				Outputs: []*transactions.Output{
					{
						Commitment:      make([]byte, types.CommitmentLen),
						EphemeralPubkey: make([]byte, PubkeyLen),
						Ciphertext:      make([]byte, CiphertextLen+1),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "tx too large",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nullifier.Bytes(),
				Validator_ID: peerIDBytes,
				Locktime:     time.Time{}.Unix(),
				Signature:    make([]byte, 1000000),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
	}
	for _, test := range tests {
		err := CheckTransactionSanity(test.tx, test.timestamp)
		if test.expectedErr == nil {
			assert.NoErrorf(t, err, "tx sanity test: %s failure", test.name)
		} else {
			assert.Equalf(t, test.expectedErr.(RuleError).ErrorCode, err.(RuleError).ErrorCode, "tx sanity test: %s: error %s", test.name, err.Error())
		}
	}
}
