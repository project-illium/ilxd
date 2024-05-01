// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"time"
)

const (
	PubkeyLen = 36

	// CiphertextLen is:
	// ScriptHashLen + AssetIDLen + SaltLen + AmountLen + StateLen + CiphertextOverhead
	CiphertextLen = 32 + 32 + 32 + 8 + 128 + 48

	MaxDocumentHashLen = 68

	MaxTransactionSize = 1000000

	MaxBlockFutureTime = time.Second * 10

	RestakePeriod = time.Hour * 24 * 7
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

	// BFNoNotification is used to signal that the subscribes should not
	// be notified about any events in the block.
	BFNoNotification

	// BFNoFlush skips flushing memory caches to disk.
	BFNoFlush

	// BFBatchCommit skips the database commit and allows committing
	// as a batch.
	BFBatchCommit

	// BFNone is a convenience value to specifically indicate no flags.
	BFNone BehaviorFlags = 0
)

// HasFlag returns whether the BehaviorFlags has the passed flag set.
func (behaviorFlags BehaviorFlags) HasFlag(flag BehaviorFlags) bool {
	return behaviorFlags&flag == flag
}

// checkBlockContext checks that the block connects to the tip of the chain and that
// the block producer exists in the validator set.
func (b *Blockchain) checkBlockContext(header *blocks.BlockHeader) error {
	tip := b.index.Tip()
	if header == nil {
		return ruleError(ErrNilHeader, "header is nil")
	}
	if header.Height <= tip.Height() {
		return ruleError(ErrDoesNotConnect, "block height less than current tip")
	}
	if header.Height > tip.Height()+1 {
		return OrphanBlockError("block is orphan")
	}
	if types.NewID(header.Parent) != tip.ID() {
		return ruleError(ErrDoesNotConnect, "block parent does not extend tip")
	}

	if header.Timestamp <= tip.Timestamp() {
		return ruleError(ErrInvalidTimestamp, "timestamp is too early")
	}
	// The block timestamp is not allowed to be too far ahead of our local clock.
	// Because this block *may* still become valid as our clock advances we will
	// mark it as an orphan which will allow us to process it again later.
	if time.Unix(header.Timestamp, 0).After(time.Now().Add(MaxBlockFutureTime)) {
		return OrphanBlockError("block timestamp is too far in the future")
	}

	producerID, err := peer.IDFromBytes(header.Producer_ID)
	if err != nil {
		return ruleError(ErrInvalidProducer, "block producer ID does not decode")
	}

	if !b.validatorSet.ValidatorExists(producerID) {
		return ruleError(ErrInvalidProducer, "block producer not in validator set")
	}
	if len(b.params.Checkpoints) > 0 && header.Height <= b.params.Checkpoints[len(b.params.Checkpoints)-1].Height {
		for _, checkpoint := range b.params.Checkpoints {
			if header.Height == checkpoint.Height && header.ID() != checkpoint.BlockID {
				return ruleError(ErrInvalidCheckpoint, "block ID does not match checkpoint")
			}
		}
	}
	return nil
}

// validateHeader validates the transaction header. No blockchain context is needed for this validation.
func (b *Blockchain) validateHeader(header *blocks.BlockHeader, flags BehaviorFlags) error {
	if header == nil {
		return ruleError(ErrNilHeader, "header is nil")
	}
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

// validateBlock validates that the block is valid according to the consensus rules.
// BLockchain context is used when validating the block as queries to the validator set,
// treasury, tx root set, etc are made.
//
// Returns the index of the transaction that was invalid or -1 if none were determined
// to be invalid.
func (b *Blockchain) validateBlock(blk *blocks.Block, flags BehaviorFlags) (int, error) {
	if err := b.validateHeader(blk.Header, flags); err != nil {
		return -1, err
	}

	if len(blk.Transactions) == 0 {
		return -1, ruleError(ErrEmptyBlock, "block contains zero transactions")
	}

	calculatedTxRoot := TransactionsMerkleRoot(blk.Transactions)

	if !bytes.Equal(calculatedTxRoot[:], blk.Header.TxRoot) {
		return -1, ruleError(ErrInvalidTxRoot, "transaction merkle root is invalid")
	}

	var (
		blockNullifiers   = make(map[types.Nullifier]bool)
		blockCoinbases    = make(map[peer.ID]bool)
		stakeTransactions = make([]*transactions.StakeTransaction, 0, len(blk.Transactions))

		treasuryBalance *types.Amount
		lastTxid        = types.NewID(make([]byte, 32))
	)

	for i, t := range blk.GetTransactions() {
		if !flags.HasFlag(BFGenesisValidation) {
			if lastTxid.Compare(t.ID()) >= 0 {
				return -1, ruleError(ErrBlockSort, "block is not sorted by txid")
			}
			lastTxid = t.ID()
		}
		if err := CheckTransactionSanity(t, time.Unix(blk.Header.Timestamp, 0)); err != nil {
			return i, err
		}
		switch tx := t.Tx.(type) {
		case *transactions.Transaction_CoinbaseTransaction:
			validatorID, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
			if err != nil {
				return i, ruleError(ErrInvalidTx, "coinbase tx validator ID does not decode")
			}
			if !flags.HasFlag(BFGenesisValidation) {
				if blockCoinbases[validatorID] {
					return i, ruleError(ErrDuplicateCoinbase, "more than one coinbase per validator")
				}
				validator, err := b.validatorSet.GetValidator(validatorID)
				if err != nil {
					return i, ruleError(ErrInvalidTx, "validator does not exist in validator set")
				}
				if types.Amount(tx.CoinbaseTransaction.NewCoins) != validator.UnclaimedCoins || tx.CoinbaseTransaction.NewCoins == 0 {
					return i, ruleError(ErrInvalidTx, "coinbase transaction creates invalid number of coins")
				}
				blockCoinbases[validatorID] = true

				if !bytes.Equal(tx.CoinbaseTransaction.Epoch_ID, b.lastEpochID.Bytes()) {
					return i, ruleError(ErrInvalidTx, "coinbase epoch ID not valid")
				}

				if tx.CoinbaseTransaction.EpochHeight != b.lastEpochHeight {
					return i, ruleError(ErrInvalidTx, "coinbase epoch height not valid")
				}
			}
		case *transactions.Transaction_StakeTransaction:
			stakeTransactions = append(stakeTransactions, tx.StakeTransaction)
			if !flags.HasFlag(BFGenesisValidation) {
				exists, err := b.txoRootSet.RootExists(types.NewID(tx.StakeTransaction.TxoRoot))
				if err != nil {
					return i, err
				}
				if !exists {
					return i, ruleError(ErrInvalidTx, "txo root does not exist in chain")
				}
				exists, err = b.nullifierSet.NullifierExists(types.NewNullifier(tx.StakeTransaction.Nullifier))
				if err != nil {
					return i, err
				}
				if exists {
					return i, ruleError(ErrDoubleSpend, "stake tx contains spent nullifier")
				}
				valID, err := peer.IDFromBytes(tx.StakeTransaction.Validator_ID)
				if err != nil {
					return i, ruleError(ErrInvalidTx, "stake tx validator ID does not decode")
				}
				validator, err := b.validatorSet.GetValidator(valID)
				if err == nil {
					stake, exists := validator.Nullifiers[types.NewNullifier(tx.StakeTransaction.Nullifier)]
					if exists {
						if stake.Blockstamp.Add(ValidatorExpiration - RestakePeriod).After(time.Unix(blk.Header.Timestamp, 0)) {
							return i, ruleError(ErrRestakeTooEarly, "restake transaction too early")
						}
					}
				}
			}
		case *transactions.Transaction_StandardTransaction:
			if flags.HasFlag(BFGenesisValidation) {
				return i, ruleError(ErrInvalidGenesis, "genesis block should only contain coinbase and stake txs")
			}
			for _, n := range tx.StandardTransaction.Nullifiers {
				nullifier := types.NewNullifier(n)
				if blockNullifiers[nullifier] {
					return i, ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}
				exists, err := b.nullifierSet.NullifierExists(nullifier)
				if err != nil {
					return i, err
				}
				if exists {
					return i, ruleError(ErrDoubleSpend, "block contains spent nullifier")
				}

				blockNullifiers[nullifier] = true
			}
			exists, err := b.txoRootSet.RootExists(types.NewID(tx.StandardTransaction.TxoRoot))
			if err != nil {
				return i, err
			}
			if !exists {
				return i, ruleError(ErrInvalidTx, "txo root does not exist in chain")
			}
		case *transactions.Transaction_MintTransaction:
			if flags.HasFlag(BFGenesisValidation) {
				return i, ruleError(ErrInvalidGenesis, "genesis block should only contain coinbase and stake txs")
			}
			for _, n := range tx.MintTransaction.Nullifiers {
				nullifier := types.NewNullifier(n)
				if blockNullifiers[nullifier] {
					return i, ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}
				exists, err := b.nullifierSet.NullifierExists(nullifier)
				if err != nil {
					return i, err
				}
				if exists {
					return i, ruleError(ErrDoubleSpend, "block contains duplicate nullifier")
				}

				blockNullifiers[nullifier] = true
			}

			exists, err := b.txoRootSet.RootExists(types.NewID(tx.MintTransaction.TxoRoot))
			if err != nil {
				return i, err
			}
			if !exists {
				return i, ruleError(ErrInvalidTx, "txo root does not exist in chain")
			}
		case *transactions.Transaction_TreasuryTransaction:
			if flags.HasFlag(BFGenesisValidation) {
				return i, ruleError(ErrInvalidGenesis, "genesis block should only contain coinbase and stake txs")
			}
			if treasuryBalance == nil {
				balance, err := dsFetchTreasuryBalance(b.ds)
				if err != nil {
					return i, err
				}
				treasuryBalance = &balance
			}

			if types.Amount(tx.TreasuryTransaction.Amount) > *treasuryBalance {
				return i, ruleError(ErrInvalidTx, "treasury tx amount exceeds treasury balance")
			}
			*treasuryBalance -= types.Amount(tx.TreasuryTransaction.Amount)
		default:
			return i, ruleError(ErrInvalidTx, "unknown transaction type")
		}
	}

	for _, stakeTx := range stakeTransactions {
		if blockNullifiers[types.NewNullifier(stakeTx.Nullifier)] {
			return -1, ruleError(ErrBlockStakeSpend, "stake created and spent in the same block")
		}
	}

	if flags.HasFlag(BFGenesisValidation) {
		if len(blk.Transactions) < 2 {
			return -1, ruleError(ErrInvalidGenesis, "genesis block must at least two txs")
		}
		coinbaseTx, ok := blk.Transactions[0].GetTx().(*transactions.Transaction_CoinbaseTransaction)
		if !ok {
			return -1, ruleError(ErrInvalidGenesis, "first genesis transaction is not a coinbase")
		}
		acc := NewAccumulator()
		for _, output := range coinbaseTx.CoinbaseTransaction.Outputs {
			acc.Insert(output.Commitment, false)
		}
		txoRoot := acc.Root().Bytes()

		totalStaked := types.Amount(0)
		for _, stakeTx := range stakeTransactions {
			if !bytes.Equal(stakeTx.TxoRoot, txoRoot) {
				return -1, ruleError(ErrInvalidGenesis, "genesis stake txoroot invalid")
			}
			totalStaked += types.Amount(stakeTx.Amount)
		}
		if totalStaked > types.Amount(coinbaseTx.CoinbaseTransaction.NewCoins) {
			return -1, ruleError(ErrInvalidGenesis, "genesis total stake larger than coinbase")
		}
	}

	if !flags.HasFlag(BFFastAdd) {
		proofValidator := NewProofValidator(b.proofCache, b.verifier)
		if err := proofValidator.Validate(blk.Transactions); err != nil {
			return -1, err
		}
		sigValidator := NewSigValidator(b.sigCache)
		if err := sigValidator.Validate(blk.Transactions); err != nil {
			return -1, err
		}
	}

	return -1, nil
}

// CheckTransactionSanity performs a sanity check on the transaction. No blockchain context
// is considered by this function.
func CheckTransactionSanity(t *transactions.Transaction, blockTime time.Time) error {
	if t.Tx == nil {
		return ruleError(ErrInvalidTx, "missing inner protobuf transaction")
	}
	switch tx := t.Tx.(type) {
	case *transactions.Transaction_CoinbaseTransaction:
		if err := validateOutputs(tx.CoinbaseTransaction.Outputs); err != nil {
			return err
		}
		_, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
		if err != nil {
			return ruleError(ErrInvalidTx, "coinbase tx validator ID does not decode")
		}
	case *transactions.Transaction_StakeTransaction:
		if tx.StakeTransaction.Nullifier == nil {
			return ruleError(ErrInvalidTx, "stake transaction missing nullifier")
		}
		_, err := peer.IDFromBytes(tx.StakeTransaction.Validator_ID)
		if err != nil {
			return ruleError(ErrInvalidTx, "coinbase tx validator ID does not decode")
		}
	case *transactions.Transaction_StandardTransaction:
		if len(tx.StandardTransaction.Nullifiers) == 0 {
			return ruleError(ErrInvalidTx, "transaction missing nullifier(s)")
		}
		if err := validateOutputs(tx.StandardTransaction.Outputs); err != nil {
			return err
		}
		if !ValidateLocktime(blockTime, tx.StandardTransaction.Locktime) {
			return ruleError(ErrInvalidTx, "transaction locktime is invalid")
		}
	case *transactions.Transaction_MintTransaction:
		if len(tx.MintTransaction.Nullifiers) == 0 {
			return ruleError(ErrInvalidTx, "transaction missing nullifier(s)")
		}
		if err := validateOutputs(tx.MintTransaction.Outputs); err != nil {
			return err
		}
		if len(tx.MintTransaction.Asset_ID) > types.AssetIDLen {
			return ruleError(ErrInvalidTx, "invalid asset ID")
		}
		if len(tx.MintTransaction.DocumentHash) > MaxDocumentHashLen {
			return ruleError(ErrInvalidTx, "mint document hash too long")
		}
		switch tx.MintTransaction.Type {
		case transactions.MintTransaction_FIXED_SUPPLY:
			if !bytes.Equal(tx.MintTransaction.Asset_ID, tx.MintTransaction.Nullifiers[0]) {
				return ruleError(ErrInvalidTx, "fixed supply mint transaction invalid assetID")
			}
		case transactions.MintTransaction_VARIABLE_SUPPLY:
			if bytes.Equal(tx.MintTransaction.Asset_ID, types.IlliumCoinID[:]) {
				return ruleError(ErrInvalidTx, "variable supply mint transaction uses illium assetID")
			}
			pubkey, err := crypto.UnmarshalPublicKey(tx.MintTransaction.MintKey)
			if err != nil {
				return ruleError(ErrInvalidTx, "invalid mint key")
			}
			rawPubkey, err := pubkey.Raw()
			if err != nil {
				return ruleError(ErrInvalidTx, "invalid mint key")
			}

			keyHash := hash.HashFunc(rawPubkey)

			if !bytes.Equal(tx.MintTransaction.Asset_ID, keyHash) {
				return ruleError(ErrInvalidTx, "variable supply mint transaction invalid assetID")
			}
		default:
			return ruleError(ErrUnknownTxEnum, "unknown mint transaction type")
		}
		if !ValidateLocktime(blockTime, tx.MintTransaction.Locktime) {
			return ruleError(ErrInvalidTx, "transaction locktime is invalid")
		}
	case *transactions.Transaction_TreasuryTransaction:
		if err := validateOutputs(tx.TreasuryTransaction.Outputs); err != nil {
			return err
		}
		if len(tx.TreasuryTransaction.ProposalHash) > MaxDocumentHashLen {
			return ruleError(ErrInvalidTx, "treasury proposal hash too long")
		}
	default:
		return ruleError(ErrInvalidTx, "unknown transaction type")
	}
	size, err := t.SerializedSize()
	if err != nil {
		return err
	}
	if size > MaxTransactionSize {
		return ruleError(ErrInvalidTx, "transaction too large")
	}
	return nil
}

// ValidateLocktime validates that the blocktime is within the locktime range
// specified by the provided precision.
func ValidateLocktime(blocktime time.Time, locktime *transactions.Locktime) bool {
	if locktime == nil || locktime.Timestamp <= 0 {
		return true
	}
	timestamp := time.Unix(locktime.Timestamp, 0)
	precisionDuration := time.Duration(locktime.Precision) * time.Second

	windowStart := timestamp.Add(-precisionDuration)
	windowEnd := timestamp.Add(precisionDuration)

	if blocktime.After(windowStart) && blocktime.Before(windowEnd) {
		return true
	}
	return false
}

// validateOutputs makes sure the output fields do not exceed a certain length. Protobuf
// does not enforce size restrictions so we have to do it here.
func validateOutputs(outputs []*transactions.Output) error {
	if len(outputs) == 0 {
		return ruleError(ErrInvalidTx, "transaction has no outputs")
	}
	for _, out := range outputs {
		if len(out.Commitment) != types.CommitmentLen {
			return ruleError(ErrInvalidTx, "invalid commitment len")
		}
	}
	return nil
}
