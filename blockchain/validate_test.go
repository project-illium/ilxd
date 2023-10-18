// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
		params:       &params.RegestParams,
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
			expectedErr: OrphanBlockError("block is orphan"),
		},
		{
			name: "height before tip",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height - 1,
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
			name: "timestamp too far in future",
			header: &blocks.BlockHeader{
				Version:     1,
				Height:      prev.Height + 1,
				Parent:      prevID[:],
				Timestamp:   time.Now().Add(MaxBlockFutureTime + time.Second*1).Unix(),
				Producer_ID: valBytes,
			},
			expectedErr: OrphanBlockError("block timestamp is too far in the future"),
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
		} else if _, ok := test.expectedErr.(OrphanBlockError); ok {
			assert.Equal(t, test.expectedErr.(OrphanBlockError), err, "block context test: %s failure", test.name)
		} else {
			assert.Equal(t, test.expectedErr.(RuleError).ErrorCode, err.(RuleError).ErrorCode, "block context test: %s failure", test.name)
		}
	}
}

func TestValidateBlock(t *testing.T) {
	ds := mock.NewMapDatastore()
	b := Blockchain{
		ds:           ds,
		proofCache:   NewProofCache(30),
		txoRootSet:   NewTxoRootSet(ds, 10),
		nullifierSet: NewNullifierSet(ds, 10),
		validatorSet: NewValidatorSet(&params.RegestParams, ds),
	}
	assert.NoError(t, dsInitTreasury(ds))
	dbtx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)
	assert.NoError(t, dsCreditTreasury(dbtx, 10000))
	assert.NoError(t, dbtx.Commit(context.Background()))

	// Build valid block
	header := randomBlockHeader(1, randomID())
	pkBytes, err := hex.DecodeString("08011240a9aa020035ffcd6b474e45667ed6cb76a8c8b570b09f7605c488a0ea881a2ed8ee319ce201a24dbef41b6480c7f6315930825ddb0476662c6afa837573ef063f")
	assert.NoError(t, err)
	sk, err := crypto.UnmarshalPrivateKey(pkBytes)
	assert.NoError(t, err)

	pid, err := peer.IDFromPublicKey(sk.GetPublic())
	assert.NoError(t, err)

	pidBytes, err := pid.Marshal()
	assert.NoError(t, err)
	header.Producer_ID = pidBytes

	block := &blocks.Block{
		Header:       header,
		Transactions: nil,
	}

	hexToBytes := func(s string) []byte {
		ret, _ := hex.DecodeString(s)
		return ret
	}

	// Hardcoded instead of random to preserve block sort order
	var (
		txoRoot      = hexToBytes("6d509adaffb1d910a13c88383bd39552673ec34d3b12043bac04ad1bde2c7bb5")
		txoRoot2     = hexToBytes("f1255b59e9b04a5248b36f44d0aa7d28c563f8de605251034452f54b4d212e67")
		nullifier    = hexToBytes("4adde2dd6a0ee6af84839962c4649551c6b0c16099f413ff01d2b7438a21a6e9")
		nullifier2   = hexToBytes("a731ba862b015ce93e4236d87bcfe0e37d156d1846dccdf3f8dd82764c970620")
		nullifier3   = hexToBytes("22ef55682643310babc449255a510523be2ef31c1fd36581bcc907b35642c6d2")
		nullifier4   = hexToBytes("32e77ef1961f1c9c5135bff6e939b7ef3b4e8c7defa61d50dca797c86f97cc91")
		validatorID  = "12D3KooWRjmdSPh7WZmbYfiRXtt1cXAfGV6Q5nTQFwknWfEh5tT2"
		validatorID2 = "12D3KooWP1qxNbQQLUFuZDfXkwPRC8p9UbVNVy3ytzaqryv8DJXz"
	)

	validatorPid, err := peer.Decode(validatorID)
	assert.NoError(t, err)
	validatorIDBytes, err := validatorPid.Marshal()
	assert.NoError(t, err)
	validatorPid2, err := peer.Decode(validatorID2)
	assert.NoError(t, err)
	validatorID2Bytes, err := validatorPid2.Marshal()
	assert.NoError(t, err)
	b.txoRootSet.cache[types.NewID(txoRoot)] = time.Now()
	b.nullifierSet.cachedEntries[types.NewNullifier(nullifier2[:])] = true

	b.validatorSet.validators[validatorPid] = &Validator{
		PeerID:         validatorPid,
		UnclaimedCoins: 10000,
		Nullifiers: map[types.Nullifier]Stake{
			types.NewNullifier(nullifier4): {
				Amount:     25,
				Blockstamp: time.Now(),
			},
		},
	}

	acc := NewAccumulator()
	acc.Insert(make([]byte, types.CommitmentLen), false)
	root := acc.Root()

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
			name: "transactions out of required order",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier3[:]},
						TxoRoot:    txoRoot[:],
					}),
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrBlockSort, ""),
		},
		{
			name: "standard transaction during genesis validation",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
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
			name: "standard transaction valid",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: nil,
		},
		{
			name: "duplicate transaction (will violate block sort rule)",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrBlockSort, ""),
		},
		{
			name: "standard transaction nullifier already in block",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: bytes.Repeat([]byte{0x11}, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrDoubleSpend, ""),
		},
		{
			name: "standard transaction nullifier already in nullifier set",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier2[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrDoubleSpend, ""),
		},
		{
			name: "standard transaction txo root doesn't exist in set",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot2[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
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
		{
			name: "stake transaction valid",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: nil,
		},
		{
			name: "stake valid restake",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier4,
						TxoRoot:      txoRoot[:],
						Amount:       25,
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header.Timestamp = time.Now().Add(ValidatorExpiration - RestakePeriod + time.Second).Unix()
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: nil,
		},
		{
			name: "stake restake too early",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier4,
						TxoRoot:      txoRoot[:],
						Amount:       25,
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header.Timestamp = time.Now().Add(ValidatorExpiration - RestakePeriod - time.Second*2).Unix()
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrRestakeTooEarly, ""),
		},
		{
			name: "stake transaction txo root doesn't exist in set",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      txoRoot2[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "staked nullifier spent in same block",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      txoRoot[:],
					}),
					transactions.WrapTransaction(&transactions.StandardTransaction{
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrBlockStakeSpend, ""),
		},
		{
			name: "mint transaction during genesis validation",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.MintTransaction{
						Type: transactions.MintTransaction_VARIABLE_SUPPLY,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
						MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
					}),
				}
				keyHash := hash.HashFunc(blk.Transactions[0].GetMintTransaction().MintKey)
				blk.Transactions[0].GetMintTransaction().Asset_ID = keyHash

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
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
			name: "mint transaction valid",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.MintTransaction{
						Type: transactions.MintTransaction_VARIABLE_SUPPLY,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
						Asset_ID:   bytes.Repeat([]byte{0x11}, types.AssetIDLen),
						MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
					}),
				}
				keyHash := hash.HashFunc(blk.Transactions[0].GetMintTransaction().MintKey)
				blk.Transactions[0].GetMintTransaction().Asset_ID = keyHash

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: nil,
		},
		{
			name: "mint transaction nullifier already in block",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.MintTransaction{
						Type: transactions.MintTransaction_VARIABLE_SUPPLY,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
						Asset_ID:   bytes.Repeat([]byte{0x12}, types.AssetIDLen),
						MintKey:    bytes.Repeat([]byte{0x12}, PubkeyLen),
					}),
					transactions.WrapTransaction(&transactions.MintTransaction{
						Type: transactions.MintTransaction_VARIABLE_SUPPLY,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Asset_ID:   bytes.Repeat([]byte{0x11}, types.AssetIDLen),
						MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot[:],
					}),
				}
				keyHash := hash.HashFunc(blk.Transactions[0].GetMintTransaction().MintKey)
				blk.Transactions[0].GetMintTransaction().Asset_ID = keyHash
				keyHash2 := hash.HashFunc(blk.Transactions[1].GetMintTransaction().MintKey)
				blk.Transactions[1].GetMintTransaction().Asset_ID = keyHash2

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrDoubleSpend, ""),
		},
		{
			name: "mint transaction nullifier already in nullifier set",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.MintTransaction{
						Type: transactions.MintTransaction_VARIABLE_SUPPLY,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Asset_ID:   bytes.Repeat([]byte{0x11}, types.AssetIDLen),
						MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
						Nullifiers: [][]byte{nullifier2[:]},
						TxoRoot:    txoRoot[:],
					}),
				}
				keyHash := hash.HashFunc(blk.Transactions[0].GetMintTransaction().MintKey)
				blk.Transactions[0].GetMintTransaction().Asset_ID = keyHash

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFNone,
			expectedErr: ruleError(ErrDoubleSpend, ""),
		},
		{
			name: "mint transaction txo root doesn't exist in set",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.MintTransaction{
						Type: transactions.MintTransaction_VARIABLE_SUPPLY,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						Asset_ID:   bytes.Repeat([]byte{0x11}, types.AssetIDLen),
						MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
						Nullifiers: [][]byte{nullifier[:]},
						TxoRoot:    txoRoot2[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
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
		{
			name: "coinbase transaction valid",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: nil,
		},
		{
			name: "coinbase transaction incorrect new coins",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     11000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase transaction zero new coins",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     0,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase transaction validator doesn't exist",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorID2Bytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "coinbase transaction duplicate validator",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10002,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrDuplicateCoinbase, ""),
		},
		{
			name: "treasury transaction during genesis validation",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.TreasuryTransaction{
						Amount: 10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						ProposalHash: make([]byte, 32),
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
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
			name: "treasury transaction valid",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.TreasuryTransaction{
						Amount: 10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						ProposalHash: make([]byte, 32),
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: nil,
		},
		{
			name: "treasury amount to large",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.TreasuryTransaction{
						Amount: 11000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						ProposalHash: make([]byte, 32),
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "treasury cumulative amount to large",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.TreasuryTransaction{
						Amount: 8000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						ProposalHash: make([]byte, 32),
					}),
					transactions.WrapTransaction(&transactions.TreasuryTransaction{
						Amount: 3000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
						ProposalHash: make([]byte, 32),
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFFastAdd,
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "valid genesis",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      root[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFGenesisValidation | BFFastAdd,
			expectedErr: nil,
		},
		{
			name: "genesis staked coins exceeds coinbase",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      root[:],
						Amount:       10001,
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFGenesisValidation | BFFastAdd,
			expectedErr: ruleError(ErrInvalidGenesis, ""),
		},
		{
			name: "genesis invalid root",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      txoRoot[:],
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFGenesisValidation | BFFastAdd,
			expectedErr: ruleError(ErrInvalidGenesis, ""),
		},
		{
			name: "genesis invalid number of txs",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFGenesisValidation | BFFastAdd,
			expectedErr: ruleError(ErrInvalidGenesis, ""),
		},
		{
			name: "genesis tx0 not coinbase",
			block: func(blk *blocks.Block) (*blocks.Block, error) {
				blk.Transactions = []*transactions.Transaction{
					transactions.WrapTransaction(&transactions.StakeTransaction{
						Validator_ID: validatorIDBytes,
						Nullifier:    nullifier,
						TxoRoot:      root[:],
					}),
					transactions.WrapTransaction(&transactions.CoinbaseTransaction{
						Validator_ID: validatorIDBytes,
						NewCoins:     10000,
						Outputs: []*transactions.Output{
							{
								Commitment: make([]byte, types.CommitmentLen),
								Ciphertext: make([]byte, CiphertextLen),
							},
						},
					}),
				}

				merkleRoot := TransactionsMerkleRoot(blk.Transactions)
				header.TxRoot = merkleRoot[:]
				header, err := signHeader(proto.Clone(header).(*blocks.BlockHeader))
				if err != nil {
					return nil, err
				}
				blk.Header = header
				return blk, nil
			},
			flags:       BFGenesisValidation | BFFastAdd,
			expectedErr: ruleError(ErrInvalidGenesis, ""),
		},
	}

	for _, test := range tests {
		blk, err := test.block(proto.Clone(block).(*blocks.Block))
		assert.NoError(t, err)
		err = b.validateBlock(blk, test.flags)
		if test.expectedErr == nil {
			assert.NoErrorf(t, err, "block validation test: %s failure", test.name)
		} else {
			_, ok := err.(RuleError)
			assert.True(t, ok)
			if ok {
				assert.Equalf(t, test.expectedErr.(RuleError).ErrorCode, err.(RuleError).ErrorCode, "block validation test: %s: error %s", test.name, err.Error())
			} else {
				fmt.Println(err, test.name)
			}
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
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
						Commitment: make([]byte, types.CommitmentLen+1),
						Ciphertext: make([]byte, CiphertextLen),
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
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "standard no nullifiers",
			tx: transactions.WrapTransaction(&transactions.StandardTransaction{
				Nullifiers: [][]byte{},
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen+1),
						Ciphertext: make([]byte, CiphertextLen),
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
				Locktime: &transactions.Locktime{
					Timestamp: time.Now().Add(time.Hour).Unix(),
					Precision: 10,
				},
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "stake invalid validator ID",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nullifier.Bytes(),
				Validator_ID: nil,
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "stake invalid nullifier",
			tx: transactions.WrapTransaction(&transactions.StakeTransaction{
				Nullifier:    nil,
				Validator_ID: peerIDBytes,
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint valid fixed supply",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{nullifier.Bytes()},
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID: hash.HashFunc(bytes.Repeat([]byte{0x11}, PubkeyLen)),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
					},
				},
				Nullifiers: [][]byte{nullifier.Bytes()},
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
			}),
			timestamp:   time.Now(),
			expectedErr: nil,
		},
		{
			name: "mint no outputs",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Outputs:    []*transactions.Output{},
				Nullifiers: [][]byte{nullifier.Bytes()},
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint no nullifiers",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers: [][]byte{},
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID:   hash.HashFunc(nullifier.Bytes()),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen+1),
						Ciphertext: make([]byte, CiphertextLen),
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
				Locktime: &transactions.Locktime{
					Timestamp: time.Now().Add(time.Hour).Unix(),
					Precision: 10,
				},
				Asset_ID: hash.HashFunc(nullifier.Bytes()),
				MintKey:  bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID:   []byte{0x01},
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "mint invalid document hash len",
			tx: transactions.WrapTransaction(&transactions.MintTransaction{
				Nullifiers:   [][]byte{nullifier.Bytes()},
				Asset_ID:     hash.HashFunc(nullifier.Bytes()),
				DocumentHash: make([]byte, MaxDocumentHashLen+1),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID:   bytes.Repeat([]byte{0x11}, types.AssetIDLen),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID:   types.IlliumCoinID[:],
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID:   bytes.Repeat([]byte{0x22}, types.AssetIDLen),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				Asset_ID:   bytes.Repeat([]byte{0x11}, types.AssetIDLen),
				MintKey:    bytes.Repeat([]byte{0x11}, PubkeyLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrUnknownTxEnum, ""),
		},
		{
			name: "treasury valid",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxDocumentHashLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
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
				ProposalHash: make([]byte, MaxDocumentHashLen+1),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen),
						Ciphertext: make([]byte, CiphertextLen),
					},
				},
			}),
			timestamp:   time.Now(),
			expectedErr: ruleError(ErrInvalidTx, ""),
		},
		{
			name: "treasury invalid commitment len",
			tx: transactions.WrapTransaction(&transactions.TreasuryTransaction{
				ProposalHash: make([]byte, MaxDocumentHashLen),
				Outputs: []*transactions.Output{
					{
						Commitment: make([]byte, types.CommitmentLen+1),
						Ciphertext: make([]byte, CiphertextLen),
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
