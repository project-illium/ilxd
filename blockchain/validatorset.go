// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/weightedrand/v2"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	// TODO: decide on a value for this
	maxTimeBetweenFlushes = time.Minute * 15

	ValidatorExpiration = time.Hour * 24 * 7 * 26
)

// setConsistencyStatus (SCS) codes are used to indicate the
// consistency status of the validator set state in the database.
type setConsistencyStatus uint8

const (
	// scsEmpty is used as a return value to indicate that no status was
	// stored.  The zero value should not be stored in the database.
	scsEmpty setConsistencyStatus = iota

	// scsConsistent indicates that the set is consistent with the last
	// flush hash stored in the database.
	scsConsistent

	// scsFlushOngoing indicates a flush is ongoing. If a node states with this
	// state it means it must have crashed in the middle of a flush.
	scsFlushOngoing

	// scsNbCodes is the number of valid consistency status codes.
	scsNbCodes

	secondsPerMonth = 2628000
)

// Weighting params
const (
	beta0      = 0.1   // Level parameter
	beta1      = -0.1  // Slope parameter
	beta2      = -0.10 // Curvature parameter
	tau        = 12.0  // Decay parameter (converted to months)
	maxStrikes = 3     // Max strikes before a validator loses stake
)

// Stake represents a given staked utxo and the time at which it
// was added to the validator set.
type Stake struct {
	Amount         types.Amount
	WeightedAmount types.Amount
	Locktime       time.Time
	Blockstamp     time.Time
}

// Validator holds all the information about a validator in the ValidatorSet
// that is needed to validate blocks.
type Validator struct {
	PeerID           peer.ID
	TotalStake       types.Amount
	WeightedStake    types.Amount
	Nullifiers       map[types.Nullifier]Stake
	UnclaimedCoins   types.Amount
	EpochBlocks      uint32
	Strikes          uint32
	CoinbasePenalty  bool
	stakeAccumulator float64
}

// Clone returns a copy of the validator
func (v *Validator) Clone() *Validator {
	ret := &Validator{
		PeerID:           v.PeerID,
		TotalStake:       v.TotalStake,
		WeightedStake:    v.WeightedStake,
		Nullifiers:       make(map[types.Nullifier]Stake),
		UnclaimedCoins:   v.UnclaimedCoins,
		stakeAccumulator: v.stakeAccumulator,
		EpochBlocks:      v.EpochBlocks,
	}
	for n, s := range v.Nullifiers {
		ret.Nullifiers[n.Clone()] = Stake{
			Amount:     s.Amount,
			Locktime:   s.Locktime,
			Blockstamp: s.Blockstamp,
		}
	}
	return ret
}

// ValidatorSet maintains the current state of the set of validators in
// the network.
//
// We maintain a memory cache and periodically flush changes to disk.
// This is done because a change to the set may cancel out a previous
// change and committing one to disk only to have to go to disk and undo
// the change would increase overhead.
type ValidatorSet struct {
	params        *params.NetworkParams
	ds            repo.Datastore
	validators    map[peer.ID]*Validator
	nullifierMap  map[types.Nullifier]*Validator
	toDelete      map[peer.ID]struct{}
	chooser       *weightedrand.Chooser[peer.ID, types.Amount]
	EpochBlocks   uint32
	lastFlush     time.Time
	notifications []NotificationCallback
	notifMtx      sync.RWMutex
	mtx           sync.RWMutex
}

// NewValidatorSet returns a new, uninitialized, ValidatorSet.
func NewValidatorSet(params *params.NetworkParams, ds repo.Datastore) *ValidatorSet {
	vs := &ValidatorSet{
		params:       params,
		ds:           ds,
		validators:   make(map[peer.ID]*Validator),
		nullifierMap: make(map[types.Nullifier]*Validator),
		toDelete:     make(map[peer.ID]struct{}),
		mtx:          sync.RWMutex{},
	}
	return vs
}

// Init initializes the validator set. We check to make sure that the set is consistent
// with the tip of the blockchain. In the event of a hard shutdown or shutdown in the middle
// of a flush, the state could be inconsistent. If this is the case, Init will attempt
// to repair the validator set. Depending on the severity of the problem, repair could
// take a while as we may need to rebuild the set from genesis.
func (vs *ValidatorSet) Init(tip *blockNode) error {
	consistencyStatus, err := dsFetchValidatorSetConsistencyStatus(vs.ds)
	if err != nil {
		return err
	}
	lastFlushHeight, err := dsFetchValidatorLastFlushHeight(vs.ds)
	if err != nil {
		return err
	}

	switch consistencyStatus {
	case scsConsistent:
		validators, err := dsFetchValidators(vs.ds)
		if err != nil {
			return err
		}
		for _, val := range validators {
			vs.validators[val.PeerID] = val
			for nullifier := range val.Nullifiers {
				vs.nullifierMap[nullifier] = val
			}
			vs.EpochBlocks += val.EpochBlocks
		}
		if lastFlushHeight == tip.Height() {
			// We're good
			break
		} else if lastFlushHeight < tip.Height() {
			// Load the missing blocks from disk and
			// apply any changes to the validator set.
			// Build the nullifier map.
			var (
				node = tip
				err  error
			)
			for {
				if node.height == lastFlushHeight+1 {
					break
				}
				node, err = node.Parent()
				if err != nil {
					return err
				}
			}
			for {
				blk, err := node.Block()
				if err != nil {
					return err
				}

				validatorReward := types.Amount(0)

				if blk.Header.Height > 0 {
					parent, err := node.Parent()
					if err != nil {
						return err
					}
					prevHeader, err := parent.Header()
					if err != nil {
						return err
					}

					prevEpoch := (prevHeader.Timestamp - vs.params.GenesisBlock.Header.Timestamp) / vs.params.EpochLength
					blkEpoch := (blk.Header.Timestamp - vs.params.GenesisBlock.Header.Timestamp) / vs.params.EpochLength

					if blkEpoch > prevEpoch {
						validatorReward = calculateNextValidatorReward(vs.params, blkEpoch)
					}
				}

				tx, err := vs.ConnectBlock(blk, validatorReward)
				if err != nil {
					return err
				}
				if err := tx.Commit(FlushPeriodic); err != nil {
					return err
				}

				if node.height == tip.height {
					break
				}
				node, err = node.Child()
				if err != nil {
					return err
				}
			}
			if err := vs.Flush(FlushRequired, tip.height); err != nil {
				return err
			}
		} else if lastFlushHeight > tip.Height() {
			// This really should never happen.
			// If we're here it's unlikely the tip node
			// has any attached children that we can use
			// to load the blocks and remove the validator
			// changes from the set. Panic?
			log.Fatal("Validator set last flush height ahead of blockchain tip")
		}
	case scsFlushOngoing:
		// Unfortunately we can't recover from this without rebuilding
		// from genesis.
		log.Warn("Reconstructing validator set after unclean shutdown. This may take a while.")
		dbtx, err := vs.ds.NewTransaction(context.Background(), false)
		if err != nil {
			return err
		}
		if err := dsDeleteValidatorSet(dbtx); err != nil {
			return err
		}
		if err := dsPutValidatorLastFlushHeight(dbtx, 0); err != nil {
			return err
		}
		if err := dbtx.Commit(context.Background()); err != nil {
			return err
		}
		if err := dsPutValidatorSetConsistencyStatus(vs.ds, scsEmpty); err != nil {
			return err
		}
		tx, err := vs.ConnectBlock(vs.params.GenesisBlock, 0)
		if err != nil {
			return err
		}
		if err := tx.Commit(FlushRequired); err != nil {
			return err
		}
		return vs.Init(tip)
	case scsEmpty:
		// Nothing to do here
	}

	b := make([]byte, 8)
	crand.Read(b)
	rand.Seed(int64(binary.BigEndian.Uint64(b)))

	choices := make([]weightedrand.Choice[peer.ID, types.Amount], 0, len(vs.validators))
	for peerID, validator := range vs.validators {
		choices = append(choices, weightedrand.NewChoice(peerID, validator.WeightedStake))
	}
	// The chooser errors and returns nil when either:
	// - The total weight exceeds a MaxInt64 (The total coins in the network
	//   won't exceed this value for 100 years).
	// - There are zero validators.
	//
	// So we just ignore the error here and let it be nil if there are zero validators.
	// In the WeightedRandomValidator() method we will check for nil before accessing it.
	vs.chooser, _ = weightedrand.NewChooser(choices...)

	return err
}

// GetValidator returns the validator given the ID.
func (vs *ValidatorSet) GetValidator(id peer.ID) (*Validator, error) {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	val, ok := vs.validators[id]
	if !ok {
		return val, errors.New("not found")
	}
	cpy := &Validator{}
	copyValidator(cpy, val)
	return cpy, nil
}

// ValidatorExists returns whether the validator exists in the set.
func (vs *ValidatorSet) ValidatorExists(id peer.ID) bool {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	_, ok := vs.validators[id]
	return ok
}

func (vs *ValidatorSet) SubscribeEvents(callback NotificationCallback) {
	vs.notifMtx.Lock()
	vs.notifications = append(vs.notifications, callback)
	vs.notifMtx.Unlock()
}

func (vs *ValidatorSet) sendNotification(peerID peer.ID, typ NotificationType) {
	vs.notifMtx.RLock()
	n := Notification{Type: typ, Data: peerID}
	for _, callback := range vs.notifications {
		callback(&n)
	}
	vs.notifMtx.RUnlock()
}

// NullifierExists returns whether or not a nullifier exists in the set.
func (vs *ValidatorSet) NullifierExists(nullifier types.Nullifier) bool {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	_, ok := vs.nullifierMap[nullifier]
	return ok
}

// TotalStaked returns the total staked by all validators.
//
// This method is safe for concurrent access.
func (vs *ValidatorSet) TotalStaked() types.Amount {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	return vs.totalStaked()
}

// totalStaked returns the total staked by all validators.
//
// This method is NOT safe for concurrent access.
func (vs *ValidatorSet) totalStaked() types.Amount {
	total := types.Amount(0)
	for _, val := range vs.validators {
		total += val.TotalStake
	}
	return total
}

// totalWeightedStake returns the total weighted staked of all validators.
//
// This method is NOT safe for concurrent access.
func (vs *ValidatorSet) totalWeightedStake() types.Amount {
	total := types.Amount(0)
	for _, val := range vs.validators {
		total += val.WeightedStake
	}
	return total
}

// ConnectBlock stages the changes to the validator set found in the block and
// returns a transaction. The transaction can then be committed to the set using
// the Commit() method.
func (vs *ValidatorSet) ConnectBlock(blk *blocks.Block, validatorReward types.Amount) (*VsTransction, error) {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	vstx := &VsTransction{
		vs:                 vs,
		updates:            make(map[peer.ID]*Validator),
		nullifiersToAdd:    make(map[types.Nullifier]peer.ID),
		nullifiersToDelete: make(map[types.Nullifier]struct{}),
		nullifiersToBan:    make(map[types.Nullifier]struct{}),
		newEpoch:           validatorReward > 0,
		blockHeight:        blk.Header.Height,
	}

	var (
		producerID peer.ID
		err        error
		blockTime  = time.Unix(blk.Header.Timestamp, 0)
	)
	if blk.Header.Height > 0 {
		producerID, err = peer.IDFromBytes(blk.Header.Producer_ID)
		if err != nil {
			return nil, err
		}
		vstx.blockProducer = producerID
	}

	for _, t := range blk.GetTransactions() {
		switch tx := t.GetTx().(type) {
		case *transactions.Transaction_CoinbaseTransaction:
			if blk.Header.Height > 0 {
				validatorID, err := peer.IDFromBytes(tx.CoinbaseTransaction.Validator_ID)
				if err != nil {
					return nil, err
				}

				valNew, ok := vstx.updates[validatorID]
				if !ok {
					valOld, ok := vs.validators[validatorID]
					if !ok {
						log.Warn("Coinbase transaction for validator not in set")
					}
					valNew = &Validator{}
					copyValidator(valNew, valOld)
				}
				if types.Amount(tx.CoinbaseTransaction.NewCoins) >= valNew.UnclaimedCoins {
					valNew.UnclaimedCoins = 0
				} else {
					valNew.UnclaimedCoins -= types.Amount(tx.CoinbaseTransaction.NewCoins)
				}
				vstx.updates[validatorID] = valNew
			}
		case *transactions.Transaction_StakeTransaction:
			validatorID, err := peer.IDFromBytes(tx.StakeTransaction.Validator_ID)
			if err != nil {
				return nil, err
			}

			valNew, ok := vstx.updates[validatorID]
			if !ok {
				valOld, ok := vs.validators[validatorID]
				if !ok {
					valNew = &Validator{
						PeerID:         validatorID,
						TotalStake:     0,
						WeightedStake:  0,
						Nullifiers:     make(map[types.Nullifier]Stake),
						UnclaimedCoins: 0,
						EpochBlocks:    0,
					}
				} else {
					valNew = &Validator{}
					copyValidator(valNew, valOld)
				}
			}
			weight := float64(1)
			timeDiff := tx.StakeTransaction.Locktime - blk.Header.Timestamp
			if timeDiff > 0 {
				locktimeMonths := float64(timeDiff) / secondsPerMonth
				weight = 1 + approximateYieldCurve(int(locktimeMonths))
			}
			if _, ok := valNew.Nullifiers[types.NewNullifier(tx.StakeTransaction.Nullifier)]; !ok {
				if weight > 1 {
					valNew.WeightedStake += types.Amount(float64(tx.StakeTransaction.Amount) * weight)
				} else {
					valNew.WeightedStake += types.Amount(tx.StakeTransaction.Amount)
				}
				valNew.TotalStake += types.Amount(tx.StakeTransaction.Amount)
			}
			valNew.Nullifiers[types.NewNullifier(tx.StakeTransaction.Nullifier)] = Stake{
				Amount:         types.Amount(tx.StakeTransaction.Amount),
				WeightedAmount: types.Amount(float64(tx.StakeTransaction.Amount) * weight),
				Locktime:       time.Unix(tx.StakeTransaction.Locktime, 0),
				Blockstamp:     blockTime,
			}
			vstx.updates[validatorID] = valNew
			vstx.nullifiersToAdd[types.NewNullifier(tx.StakeTransaction.Nullifier)] = validatorID
		case *transactions.Transaction_StandardTransaction:
			for _, nullifier := range tx.StandardTransaction.Nullifiers {
				valOld, ok := vs.nullifierMap[types.NewNullifier(nullifier)]
				if !ok {
					continue
				}
				v, ok := vstx.updates[valOld.PeerID]
				if ok {
					valOld = v
				}
				valNew := &Validator{}
				copyValidator(valNew, valOld)
				stake, ok := valNew.Nullifiers[types.NewNullifier(nullifier)]
				if !ok {
					return nil, errors.New("nullifier not found with validator")
				}

				valNew.WeightedStake -= stake.WeightedAmount
				valNew.TotalStake -= stake.Amount
				delete(valNew.Nullifiers, types.NewNullifier(nullifier))
				vstx.nullifiersToDelete[types.NewNullifier(nullifier)] = struct{}{}
				vstx.updates[valNew.PeerID] = valNew
			}
		case *transactions.Transaction_MintTransaction:
			for _, nullifier := range tx.MintTransaction.Nullifiers {
				valOld, ok := vs.nullifierMap[types.NewNullifier(nullifier)]
				if !ok {
					continue
				}
				v, ok := vstx.updates[valOld.PeerID]
				if ok {
					valOld = v
				}
				valNew := &Validator{}
				copyValidator(valNew, valOld)
				stake, ok := valNew.Nullifiers[types.NewNullifier(nullifier)]
				if !ok {
					return nil, errors.New("nullifier not found with validator")
				}
				valNew.WeightedStake -= stake.WeightedAmount
				valNew.TotalStake -= stake.Amount
				delete(valNew.Nullifiers, types.NewNullifier(nullifier))
				vstx.nullifiersToDelete[types.NewNullifier(nullifier)] = struct{}{}
				vstx.updates[valNew.PeerID] = valNew
			}
		}
	}

	totalWeightedStake := vs.totalWeightedStake()
	if validatorReward > 0 {
		for _, valOld := range vs.validators {
			valNew, ok := vstx.updates[valOld.PeerID]
			if !ok {
				valNew = &Validator{}
				copyValidator(valNew, valOld)
			}

			expectedBlocks := valNew.stakeAccumulator
			if expectedBlocks < 1 {
				expectedBlocks = 1
			}
			valTotal := types.Amount(0)
			for nullifier, stake := range valNew.Nullifiers {
				timeSinceStake := blockTime.Sub(stake.Blockstamp)

				if timeSinceStake >= ValidatorExpiration {
					vstx.nullifiersToDelete[nullifier] = struct{}{}
					delete(valNew.Nullifiers, nullifier)
					continue
				}

				epochLength := float64(vs.params.EpochLength)
				if timeSinceStake.Seconds() >= epochLength {
					valTotal += stake.WeightedAmount
				} else {
					valTotal += types.Amount(float64(stake.WeightedAmount) * (float64(timeSinceStake.Seconds()) / epochLength))
				}
			}
			if valTotal > 0 {
				valNew.UnclaimedCoins = valNew.UnclaimedCoins + types.Amount(float64(validatorReward)*(float64(valTotal)/float64(totalWeightedStake)))
			}

			if valNew.CoinbasePenalty {
				valNew.UnclaimedCoins = 0
			}

			if valNew.EpochBlocks < blockProductionFloor(float64(vs.EpochBlocks), expectedBlocks/float64(vs.EpochBlocks)) {
				valNew.UnclaimedCoins = 0
			}

			valNew.EpochBlocks = 0
			valNew.stakeAccumulator = 0
			valNew.CoinbasePenalty = false
			vstx.updates[valNew.PeerID] = valNew
		}
	}

	if blk.Header.Height > 0 {
		blockProducer, ok := vs.validators[producerID]
		if ok {
			producerNew, ok := vstx.updates[blockProducer.PeerID]
			if !ok {
				producerNew = &Validator{}
				copyValidator(producerNew, blockProducer)
			}

			producerNew.EpochBlocks++
			expectedBlocks := blockProducer.stakeAccumulator
			if expectedBlocks < 1 {
				expectedBlocks = 1
			}
			maxBlocks := blockProductionLimit(float64(vs.EpochBlocks+1), expectedBlocks/float64(vs.EpochBlocks+1))
			if producerNew.EpochBlocks > maxBlocks {
				producerNew.CoinbasePenalty = true
				producerNew.Strikes++

				if producerNew.Strikes >= maxStrikes {
					for nullifier := range producerNew.Nullifiers {
						vstx.nullifiersToBan[nullifier] = struct{}{}
						vstx.nullifiersToDelete[nullifier] = struct{}{}
						delete(producerNew.Nullifiers, nullifier)
					}
				}
			}
			vstx.updates[blockProducer.PeerID] = producerNew
		}
	}
	return vstx, nil
}

// WeightedRandomValidator returns a validator weighted by their current stake.
//
// NOTE: If there are no validators then "" will be returned for the peer ID.
func (vs *ValidatorSet) WeightedRandomValidator() peer.ID {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	if vs.totalWeightedStake() == 0 {
		return ""
	}

	return vs.chooser.Pick()
}

// BlockProductionLimit returns the maximum blocks that a validator can produce without losing
// this coinbase. This is based on the current snapshot state of the set and changes every
// block.
func (vs *ValidatorSet) BlockProductionLimit(validatorID peer.ID) (uint32, uint32, error) {
	vs.mtx.RLock()
	defer vs.mtx.RUnlock()

	val, ok := vs.validators[validatorID]
	if !ok {
		return 0, 0, errors.New("not found")
	}
	if vs.EpochBlocks == 0 {
		return val.EpochBlocks, 1, nil
	}
	expectedBlocks := val.stakeAccumulator
	if expectedBlocks < 1 {
		expectedBlocks = 1
	}
	return val.EpochBlocks, blockProductionLimit(float64(vs.EpochBlocks), expectedBlocks/float64(vs.EpochBlocks)), nil
}

// Flush flushes changes from the memory cache to disk.
//
// This method is safe for concurrent access.
func (vs *ValidatorSet) Flush(mode flushMode, chainHeight uint32) error {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()

	return vs.flush(mode, chainHeight)
}

// flush flushes changes from the memory cache to disk.
//
// This method is NOT safe for concurrent access.
func (vs *ValidatorSet) flush(mode flushMode, chainHeight uint32) error {
	switch mode {
	case FlushRequired:
		return vs.flushToDisk(chainHeight)
	case FlushPeriodic:
		if vs.lastFlush.Add(maxTimeBetweenFlushes).Before(time.Now()) {
			return vs.flushToDisk(chainHeight)
		}
		return nil
	case FlushNop:
		return nil
	default:
		return fmt.Errorf("unsupported flushmode for the validator set")
	}
}

func (vs *ValidatorSet) flushToDisk(chainHeight uint32) error {
	if err := dsPutValidatorSetConsistencyStatus(vs.ds, scsFlushOngoing); err != nil {
		return err
	}
	dbtx, err := vs.ds.NewTransaction(context.Background(), false)
	if err != nil {
		return err
	}
	defer dbtx.Discard(context.Background())
	for _, val := range vs.validators {
		if err := dsPutValidator(dbtx, val); err != nil {
			return err
		}
	}
	for id := range vs.toDelete {
		if err := dsDeleteValidator(dbtx, id); err != nil {
			return err
		}
	}

	if err := dsPutValidatorLastFlushHeight(dbtx, chainHeight); err != nil {
		return err
	}

	if err := dbtx.Commit(context.Background()); err != nil {
		return err
	}

	if err := dsPutValidatorSetConsistencyStatus(vs.ds, scsConsistent); err != nil {
		return err
	}

	vs.toDelete = make(map[peer.ID]struct{})

	vs.lastFlush = time.Now()

	return nil
}

type VsTransction struct {
	vs                 *ValidatorSet
	updates            map[peer.ID]*Validator
	nullifiersToAdd    map[types.Nullifier]peer.ID
	nullifiersToDelete map[types.Nullifier]struct{}
	nullifiersToBan    map[types.Nullifier]struct{}

	newEpoch      bool
	blockHeight   uint32
	blockProducer peer.ID
}

func (tx *VsTransction) Commit(flushMode flushMode) error {
	for _, val := range tx.updates {
		if _, ok := tx.vs.validators[val.PeerID]; !ok {
			tx.vs.sendNotification(val.PeerID, NTAddValidator)
		}
		tx.vs.validators[val.PeerID] = val
	}

	for nullifier, peerID := range tx.nullifiersToAdd {
		val := tx.vs.validators[peerID]
		tx.vs.nullifierMap[nullifier] = val
	}

	for nullifier := range tx.nullifiersToDelete {
		delete(tx.vs.nullifierMap, nullifier)
	}

	for _, val := range tx.updates {
		if len(val.Nullifiers) == 0 {
			tx.vs.toDelete[val.PeerID] = struct{}{}
			delete(tx.vs.validators, val.PeerID)
			tx.vs.sendNotification(val.PeerID, NTRemoveValidator)
		}
	}

	if len(tx.updates) > 0 || len(tx.nullifiersToAdd) > 0 || len(tx.nullifiersToDelete) > 0 {
		choices := make([]weightedrand.Choice[peer.ID, types.Amount], 0, len(tx.vs.validators))
		for peerID, validator := range tx.vs.validators {
			choices = append(choices, weightedrand.NewChoice(peerID, validator.WeightedStake))
		}
		tx.vs.chooser, _ = weightedrand.NewChooser(choices...)
	}

	if tx.blockHeight > 0 {
		totalWeightedStake := tx.vs.totalWeightedStake()
		for _, val := range tx.vs.validators {
			val.stakeAccumulator += float64(val.WeightedStake) / float64(totalWeightedStake)
		}
	}

	if tx.newEpoch {
		tx.vs.EpochBlocks = 0
	}
	tx.vs.EpochBlocks++

	return tx.vs.flush(flushMode, tx.blockHeight)
}

func (tx *VsTransction) NullifiersToBan() []types.Nullifier {
	nullifiers := make([]types.Nullifier, 0, len(tx.nullifiersToBan))
	for n := range tx.nullifiersToBan {
		nullifiers = append(nullifiers, n)
	}
	return nullifiers
}

// Six standard deviations from the expected number of blocks.
func blockProductionLimit(EpochBlocks float64, stakePercentage float64) uint32 {
	x := float64(EpochBlocks * stakePercentage)
	y := float64(math.Sqrt(EpochBlocks*stakePercentage*(1-stakePercentage)) * 6)
	if stakePercentage == 1 {
		y = 1
	}
	return uint32(x + y)
}

// Six standard deviations from the expected number of blocks.
func blockProductionFloor(EpochBlocks float64, stakePercentage float64) uint32 {
	expectedBlocks := uint32(EpochBlocks * stakePercentage)
	dev := uint32(math.Sqrt(EpochBlocks*stakePercentage*(1-stakePercentage)) * 6)
	if dev > expectedBlocks {
		return 0
	}
	return (expectedBlocks - dev) / 3
}

// Returns a value for weight for the given maturity (in months) that approximates what you would get with a typical bond yield curve.
func approximateYieldCurve(timeToMaturity int) float64 {
	timeToMaturityInYears := float64(timeToMaturity) / 12.0
	yield := beta0 + beta1*(1-math.Exp(-timeToMaturityInYears/tau))/(timeToMaturityInYears/tau) + beta2*((1-math.Exp(-timeToMaturityInYears/tau))/(timeToMaturityInYears/tau)-math.Exp(-timeToMaturityInYears/tau))
	return yield * 1000
}

func copyValidator(dest *Validator, src *Validator) {
	dest.PeerID = src.PeerID
	dest.TotalStake = src.TotalStake
	dest.WeightedStake = src.WeightedStake
	dest.EpochBlocks = src.EpochBlocks
	dest.UnclaimedCoins = src.UnclaimedCoins
	dest.CoinbasePenalty = src.CoinbasePenalty
	dest.Strikes = src.Strikes
	dest.stakeAccumulator = src.stakeAccumulator
	dest.Nullifiers = make(map[types.Nullifier]Stake)
	for k, v := range src.Nullifiers {
		dest.Nullifiers[k.Clone()] = Stake{
			Amount:         v.Amount,
			WeightedAmount: v.WeightedAmount,
			Locktime:       v.Locktime,
			Blockstamp:     v.Blockstamp,
		}
	}
}
