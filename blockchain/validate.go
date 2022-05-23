// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/wallet"
	"time"
)

const (
	PubkeyLen = 32

	CipherTextLen = 176

	MaxProposalHashLen = 68

	MaxTransactionSize = 1000000
)

// BehaviorFlags is a bitmask defining tweaks to the normal behavior when
// performing chain processing and consensus rules checks.
type BehaviorFlags uint8

const (
	// BFFastAdd may be set to indicate that several checks can be avoided
	// for the block since it is already known to fit into the chain due to
	// already proving it correct links into the chain up to a known
	// checkpoint.  This is primarily used for headers-first mode.
	BFFastAdd BehaviorFlags = 1 << iota

	// BFNoDupBlockCheck signals if the block should skip existence
	// checks.
	BFNoDupBlockCheck

	// BFNone is a convenience value to specifically indicate no flags.
	BFNone BehaviorFlags = 0
)

// HasFlag returns whether the BehaviorFlags has the passed flag set.
func (behaviorFlags BehaviorFlags) HasFlag(flag BehaviorFlags) bool {
	return behaviorFlags&flag == flag
}

func (b *Blockchain) validateHeader(header *blocks.BlockHeader, flags BehaviorFlags) error {
	prevBlock, err := b.index.GetNodeByID(types.NewID(header.Parent))
	if err != nil {
		return ruleError(ErrDoesNotConnect, "parent block not found in chain")
	}

	if header.Height != prevBlock.height+1 {
		return ruleError(ErrInvalidHeight, "header height is invalid")
	}

	prevHeader, err := prevBlock.Header()
	if err != nil {
		return err
	}

	if header.Timestamp <= prevHeader.Timestamp {
		return ruleError(ErrInvalidTimestamp, "timestamp is too early")
	}

	producerID, err := peer.IDFromBytes(header.Producer_ID)
	if err != nil {
		return ruleError(ErrInvalidProducer, "block producer ID does not decode")
	}

	producerPubkey, err := producerID.ExtractPublicKey()
	if err != nil {
		return ruleError(ErrInvalidProducer, "block producer pubkey invalid")
	}

	if !flags.HasFlag(BFFastAdd) {
		sigHash, err := header.SigHash()
		if err != nil {
			return err
		}

		valid, err := producerPubkey.Verify(sigHash, header.Signature)
		if !valid {
			return ruleError(ErrInvalidHeaderSignature, "invalid signature in header")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Blockchain) validateBlock(blk *blocks.Block, flags BehaviorFlags) error {
	if err := b.validateHeader(blk.Header, flags); err != nil {
		return err
	}

	if len(blk.Transactions) == 0 {
		return ruleError(ErrEmptyBlock, "block contains zero transactions")
	}

	txids := make([][]byte, 0, len(blk.Transactions))
	for _, tx := range blk.Transactions {
		txids = append(txids, tx.ID().Bytes())
	}

	merkles := BuildMerkleTreeStore(txids)
	calculatedTxRoot := merkles[len(merkles)-1]

	if !bytes.Equal(calculatedTxRoot, blk.Header.TxRoot) {
		return ruleError(ErrInvalidTxRoot, "transaction merkle root is invalid")
	}

	var (
		blockNullifiers   = make(map[types.Nullifier]bool)
		stakeTransactions []*transactions.StakeTransaction
	)

	for _, t := range blk.GetTransactions() {
		switch tx := t.Tx.(type) {
		case *transactions.Transaction_CoinbaseTransaction:
			if err := validateOutputs(tx.CoinbaseTransaction.Outputs); err != nil {
				return err
			}

			validatorID, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
			if err != nil {
				return ruleError(ErrInvalidTx, "coinbase validator ID does not decode")
			}

			validatorPubkey, err := validatorID.ExtractPublicKey()
			if err != nil {
				return ruleError(ErrInvalidTx, "coinbase validator pubkey invalid")
			}

			if !flags.HasFlag(BFFastAdd) {
				sigHash, err := tx.CoinbaseTransaction.SigHash()
				if err != nil {
					return err
				}
				valid, err := validatorPubkey.Verify(sigHash, tx.CoinbaseTransaction.Signature)
				if !valid {
					return ruleError(ErrInvalidTx, "invalid signature on coinbase tx")
				}
				if err != nil {
					return err
				}
			}
			// TODO: validate amount against validator set
		case *transactions.Transaction_StakeTransaction:
			if tx.StakeTransaction.Nullifier == nil {
				return ruleError(ErrInvalidTx, "stake transaction missing nullifier")
			}
			stakeTransactions = append(stakeTransactions, tx.StakeTransaction)
			validatorID, err := peer.IDFromBytes(tx.StakeTransaction.Validator_ID)
			if err != nil {
				return ruleError(ErrInvalidTx, "stake tx validator ID does not decode")
			}

			validatorPubkey, err := validatorID.ExtractPublicKey()
			if err != nil {
				return ruleError(ErrInvalidTx, "stake tx validator pubkey invalid")
			}

			if !flags.HasFlag(BFFastAdd) {
				sigHash, err := tx.StakeTransaction.SigHash()
				if err != nil {
					return err
				}
				valid, err := validatorPubkey.Verify(sigHash, tx.StakeTransaction.Signature)
				if !valid {
					return ruleError(ErrInvalidTx, "invalid signature on stake tx")
				}
				if err != nil {
					return err
				}
			}
		case *transactions.Transaction_StandardTransaction:
			if len(tx.StandardTransaction.Nullifiers) == 0 {
				return ruleError(ErrInvalidTx, "transaction missing nullifier(s)")
			}
			if err := validateOutputs(tx.StandardTransaction.Outputs); err != nil {
				return err
			}
			for _, n := range tx.StandardTransaction.Nullifiers {
				nullifier := types.NewNullifier(n)
				if blockNullifiers[nullifier] {
					return ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}
				exists, err := b.ns.NullifierExists(nullifier)
				if err != nil {
					return err
				}
				if exists {
					return ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}

				blockNullifiers[nullifier] = true
			}
			// TODO: validate tx root
		case *transactions.Transaction_MintTransaction:
			if len(tx.MintTransaction.Nullifiers) == 0 {
				return ruleError(ErrInvalidTx, "transaction missing nullifier(s)")
			}
			if err := validateOutputs(tx.MintTransaction.Outputs); err != nil {
				return err
			}
			for _, n := range tx.MintTransaction.Nullifiers {
				nullifier := types.NewNullifier(n)
				if blockNullifiers[nullifier] {
					return ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}
				exists, err := b.ns.NullifierExists(nullifier)
				if err != nil {
					return err
				}
				if exists {
					return ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}

				blockNullifiers[nullifier] = true
			}
			switch tx.MintTransaction.Type {
			case transactions.MintTransaction_FIXED_SUPPLY:
				expectedAssetID := hash.HashFunc(tx.MintTransaction.Nullifiers[0])
				if !bytes.Equal(tx.MintTransaction.Asset_ID, expectedAssetID) {
					return ruleError(ErrInvalidTx, "fixed supply mint transaction invalid assetID")
				}
			case transactions.MintTransaction_VARIABLE_SUPPLY:
				if !bytes.Equal(tx.MintTransaction.Asset_ID, tx.MintTransaction.MintKey) {
					return ruleError(ErrInvalidTx, "variable supply mint transaction invalid assetID")
				}
			default:
				return ruleError(ErrUnknownTxEnum, "unknown mint transaction type")
			}

			mintPubkey, err := crypto.UnmarshalPublicKey(tx.MintTransaction.MintKey)
			if err != nil {
				return ruleError(ErrInvalidTx, "mint tx mint pubkey invalid")
			}

			if !flags.HasFlag(BFFastAdd) {
				sigHash, err := tx.MintTransaction.SigHash()
				if err != nil {
					return err
				}
				valid, err := mintPubkey.Verify(sigHash, tx.MintTransaction.Signature)
				if !valid {
					return ruleError(ErrInvalidTx, "invalid signature on mint tx")
				}
				if err != nil {
					return err
				}
			}
			// TODO: validate tx root
		case *transactions.Transaction_TreasuryTransaction:
			if err := validateOutputs(tx.TreasuryTransaction.Outputs); err != nil {
				return err
			}
			if len(tx.TreasuryTransaction.ProposalHash) > MaxProposalHashLen {
				return ruleError(ErrInvalidTx, "treasury proposal hash too long")
			}
			// TODO: validate treasury amount
		}
		size, err := t.SerializedSize()
		if err != nil {
			return err
		}
		if size > MaxTransactionSize {
			return ruleError(ErrInvalidTx, "transaction too large")
		}
	}

	for _, stakeTx := range stakeTransactions {
		if blockNullifiers[types.NewNullifier(stakeTx.Nullifier)] {
			return ruleError(ErrBlockStakeSpend, "stake created and spent in the same block")
		}
	}

	if !flags.HasFlag(BFFastAdd) {
		validator := NewProofValidator(time.Unix(blk.Header.Timestamp, 0))
		if err := validator.Validate(blk.Transactions); err != nil {
			return err
		}
	}

	return nil
}

func validateOutputs(outputs []*transactions.Output) error {
	for _, out := range outputs {
		if len(out.Commitment) != wallet.CommitmentLen {
			return ruleError(ErrInvalidTx, "invalid commitment")
		}
		if len(out.EphemeralPubkey) != PubkeyLen {
			return ruleError(ErrInvalidTx, "ephem pubkey invalid len")
		}
		if len(out.Ciphertext) != CipherTextLen {
			return ruleError(ErrInvalidTx, "ciphertext invalid len")
		}
	}
	return nil
}
