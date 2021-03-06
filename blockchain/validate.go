// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
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

	// BFNoValidation is used to signal that this block has already been
	// validated and is known to be good and does not need to be validated
	// again.
	BFNoValidation

	// BFGenesisValidation is the flag used to validate the genesis block
	// using special validation rules.
	BFGenesisValidation

	// BFNone is a convenience value to specifically indicate no flags.
	BFNone BehaviorFlags = 0
)

// HasFlag returns whether the BehaviorFlags has the passed flag set.
func (behaviorFlags BehaviorFlags) HasFlag(flag BehaviorFlags) bool {
	return behaviorFlags&flag == flag
}

func (b *Blockchain) checkBlockContext(header *blocks.BlockHeader) error {
	tip := b.index.Tip()
	if header.Height != tip.Height()+1 {
		return ruleError(ErrDoesNotConnect, "block does not extend tip")
	}
	if types.NewID(header.Parent) != tip.ID() {
		return ruleError(ErrDoesNotConnect, "block does not extend tip")
	}
	prevHeader, err := tip.Header()
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

	if !b.validatorSet.ValidatorExists(producerID) {
		return ruleError(ErrInvalidProducer, "block producer not in validator set")
	}

	// TODO: verifying a block ID at a checkpoint height matches the checkpoint ID will be done here.
	return nil
}

func (b *Blockchain) validateHeader(header *blocks.BlockHeader, flags BehaviorFlags) error {
	if !flags.HasFlag(BFGenesisValidation) {
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
		stakeTransactions = make([]*transactions.StakeTransaction, 0, len(blk.Transactions))
		sigsToValidate    = make([]*transactions.Transaction, 0, len(blk.Transactions))
	)

	for _, t := range blk.GetTransactions() {
		switch tx := t.Tx.(type) {
		case *transactions.Transaction_CoinbaseTransaction:
			if err := validateOutputs(tx.CoinbaseTransaction.Outputs); err != nil {
				return err
			}
			sigsToValidate = append(sigsToValidate, t)
			validatorID, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
			if err != nil {
				return ruleError(ErrInvalidTx, "coinbase tx validator ID does not decode")
			}
			if !flags.HasFlag(BFGenesisValidation) {
				validator, err := b.validatorSet.GetValidator(validatorID)
				if err != nil {
					return ruleError(ErrInvalidTx, "validator does not exist in validator set")
				}
				if tx.CoinbaseTransaction.NewCoins > validator.unclaimedCoins {
					return ruleError(ErrInvalidTx, "coinbase transaction creates too many coins")
				}
			}
		case *transactions.Transaction_StakeTransaction:
			if tx.StakeTransaction.Nullifier == nil {
				return ruleError(ErrInvalidTx, "stake transaction missing nullifier")
			}
			stakeTransactions = append(stakeTransactions, tx.StakeTransaction)
			sigsToValidate = append(sigsToValidate, t)
			if !flags.HasFlag(BFGenesisValidation) {
				exists, err := b.txoRootSet.Exists(types.NewID(tx.StakeTransaction.TxoRoot))
				if err != nil {
					return err
				}
				if !exists {
					return ruleError(ErrInvalidTx, "txo root does not exist in chain")
				}
			}
		case *transactions.Transaction_StandardTransaction:
			if flags.HasFlag(BFGenesisValidation) {
				return ruleError(ErrInvalidGenesis, "genesis block should only contain coinbase and stake txs")
			}
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
				exists, err := b.nullifierSet.NullifierExists(nullifier)
				if err != nil {
					return err
				}
				if exists {
					return ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}

				blockNullifiers[nullifier] = true
			}
			exists, err := b.txoRootSet.Exists(types.NewID(tx.StandardTransaction.TxoRoot))
			if err != nil {
				return err
			}
			if !exists {
				return ruleError(ErrInvalidTx, "txo root does not exist in chain")
			}
		case *transactions.Transaction_MintTransaction:
			if flags.HasFlag(BFGenesisValidation) {
				return ruleError(ErrInvalidGenesis, "genesis block should only contain coinbase and stake txs")
			}
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
				exists, err := b.nullifierSet.NullifierExists(nullifier)
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

			sigsToValidate = append(sigsToValidate, t)
			exists, err := b.txoRootSet.Exists(types.NewID(tx.MintTransaction.TxoRoot))
			if err != nil {
				return err
			}
			if !exists {
				return ruleError(ErrInvalidTx, "txo root does not exist in chain")
			}
		case *transactions.Transaction_TreasuryTransaction:
			if flags.HasFlag(BFGenesisValidation) {
				return ruleError(ErrInvalidGenesis, "genesis block should only contain coinbase and stake txs")
			}
			if err := validateOutputs(tx.TreasuryTransaction.Outputs); err != nil {
				return err
			}
			if len(tx.TreasuryTransaction.ProposalHash) > MaxProposalHashLen {
				return ruleError(ErrInvalidTx, "treasury proposal hash too long")
			}

			balance, err := dsFetchTreasuryBalance(b.ds)
			if err != nil {
				return err
			}
			if tx.TreasuryTransaction.Amount > balance {
				return ruleError(ErrInvalidTx, "treasury tx amount exceeds treasury balance")
			}
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

	if flags.HasFlag(BFGenesisValidation) {
		if len(blk.Transactions) < 2 {
			return ruleError(ErrInvalidGenesis, "genesis block must at least two txs")
		}
		coinbaseTx, ok := blk.Transactions[0].GetTx().(*transactions.Transaction_CoinbaseTransaction)
		if !ok {
			return ruleError(ErrInvalidGenesis, "first genesis transaction is not a coinbase")
		}
		acc := NewAccumulator()
		for _, output := range coinbaseTx.CoinbaseTransaction.Outputs {
			acc.Insert(output.Commitment, false)
		}
		txoRoot := acc.Root().Bytes()

		for _, stakeTx := range stakeTransactions {
			if !bytes.Equal(stakeTx.TxoRoot, txoRoot) {
				return ruleError(ErrInvalidGenesis, "genesis stake txoroot invalid")
			}
		}
	}

	if !flags.HasFlag(BFFastAdd) {
		proofValidator := NewProofValidator(time.Unix(blk.Header.Timestamp, 0), b.proofCache)
		if err := proofValidator.Validate(blk.Transactions); err != nil {
			return err
		}
		sigValidator := NewSigValidator(b.sigCache)
		if err := sigValidator.Validate(blk.Transactions); err != nil {
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
