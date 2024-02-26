// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/project-illium/ilxd/types"
	"time"
)

// Status is the status of consensus on a particular target
type Status int

// String returns the String representation of the Status
func (s Status) String() string {
	switch s {
	case 0:
		return "Rejected"
	case 1:
		return "NotPreferred"
	case 2:
		return "Preferred"
	case 3:
		return "Finalized"

	}
	return ""
}

const (
	// StatusRejected means the target has been rejected. Note that blocks are
	// only considered rejected if a competing block has been finalized. If the
	// confidence is a 'no' result has crossed the finalization score, it will
	// remain in the StatusNotPreferred state until a competing block has been
	// finalized.
	StatusRejected Status = iota

	// StatusNotPreferred means the target is not currently preferred by the node
	StatusNotPreferred

	// StatusPreferred means the target is currently preferred by the node
	StatusPreferred

	// StatusFinalized means the target has been finalized in the affirmative.
	StatusFinalized
)

// Result represents the result of a vote record operation
type Result uint8

const (
	// ResultNoChange means the vote did not produce any change
	// in the record state.
	ResultNoChange = iota
	// ResultFlipped means the preference of the record has flipped
	// as a result of the vote.
	ResultFlipped
	// ResultFinalized means the last vote caused a finalization
	ResultFinalized
)

// BlockChoice represents a choice of blocks at a given height in the
// chain.
//
// At every height there can be one or more blocks to chose from. This
// object tracks them all and makes a selection based on the recorded
// votes.
type BlockChoice struct {
	height           uint32
	bitRecord        *BitVoteRecord
	blockVotes       map[types.ID]*BlockVoteRecord
	inflightRequests int
	timestamp        time.Time
	totalVotes       int
}

// NewBlockChoice returns a new BlockChoice for this height
func NewBlockChoice(height uint32) *BlockChoice {
	return &BlockChoice{
		height:     height,
		bitRecord:  &BitVoteRecord{},
		blockVotes: make(map[types.ID]*BlockVoteRecord),
		timestamp:  time.Now(),
	}
}

// GetPreference returns the current block preference at this height
// or a zero ID if there is no acceptable choice.
func (bc *BlockChoice) GetPreference() types.ID {
	for id, rec := range bc.blockVotes {
		if rec.isPreferred() {
			return id.Clone()
		}
	}
	return types.ID{}
}

// HasFinalized returns whether a block at this height has finalized.
func (bc *BlockChoice) HasFinalized() bool {
	for _, rec := range bc.blockVotes {
		if rec.Status() == StatusFinalized {
			return true
		}
	}
	return false
}

// VotesNeededToFinalize returns the minimum number of votes needed
// to finalize a block given the current distribution of votes. More
// votes may ultimately be needed but this can be used to throttle
// inflight requests.
func (bc *BlockChoice) VotesNeededToFinalize() int {
	max := MaxInflightPoll
	for _, rec := range bc.blockVotes {
		confidence := rec.getConfidence()
		if MaxInflightPoll-int(confidence) < max {
			max = FinalizationScore - int(confidence)
		}
	}
	return max
}

// HasBlock returns whether a block is currently saved at this height
func (bc *BlockChoice) HasBlock(id types.ID) bool {
	_, ok := bc.blockVotes[id]
	return ok
}

// AddNewBlock adds a new block at this height. If there currently is no
// preference, and if this block is acceptable, it will be selected as
// the new preference.
func (bc *BlockChoice) AddNewBlock(blockID types.ID, isAcceptable bool) {
	havePreferred := false
	for _, record := range bc.blockVotes {
		if record.isPreferred() {
			havePreferred = true
			break
		}
	}

	preferred := false
	if !havePreferred && isAcceptable && bc.bitRecord.CompareBits(blockID) {
		preferred = true
		if bc.bitRecord.getConfidence() == 0 {
			bc.bitRecord.SetActiveBit(getBit(blockID, bc.bitRecord.activeBit) == 1)
		}
	}

	bc.blockVotes[blockID] = &BlockVoteRecord{
		acceptable: isAcceptable,
		confidence: boolToUint16(preferred),
	}
}

// RecordVote records a vote for this height. If the vote is a block ID
// then a YES will be recorded for that block and a NO for all other
// conflicting blocks. If it is a ZERO ID then neither YES nor NO will
// be recorded.
func (bc *BlockChoice) RecordVote(voteID types.ID) (types.ID, bool) {
	bc.totalVotes++

	// Set the vote to unknown if the voteID is all zeros.
	v1, v2 := byte(0x01), byte(0x00)
	if voteID.Compare(types.ID{}) == 0 {
		v1, v2 = 0x80, 0x80
	}

	// Record the new YES block vote and check for finalization
	record, ok := bc.blockVotes[voteID]
	if ok {
		if record.RecordVote(v1) == ResultFinalized {
			return voteID, true
		}
	}

	// Iterate over all other blocks and record a NO vote.
	// We need to check for (YES) finalization here because
	// even if the last vote is NO, there could still be
	// enough YES votes to increment the confidence counter
	// and cause a finalization.
	for id, record := range bc.blockVotes {
		if id != voteID {
			if record.RecordVote(v2) == ResultFinalized {
				return id, true
			}
		}
	}

	// Record the bit vote based on the voteID and check to
	// see if our bit preference either flipped or finalized.
	var (
		newPreferred *types.ID
		reselect     bool
	)
	result := bc.bitRecord.RecordVote(voteID)
	if result != ResultNoChange {
		var currentPreference types.ID
		for id, record := range bc.blockVotes {
			if record.isPreferred() {
				currentPreference = id
				break
			}
		}
		if result == ResultFinalized {
			// The current preference matches the newly finalized bits, so we
			// don't need to do anything.
			if currentPreference.Compare(types.ID{}) != 0 && bc.bitRecord.CompareBits(currentPreference) {
				return types.ID{}, false
			}

			// Loop through the existing records and see if we can't find one that matches
			// the newly finalized bits.
			//
			// This should never happen. If we correctly flipped our block preference when
			// the bit preference flipped our block preference should be correct when the
			// bit finalizes. We'll pick a new block here just in case.
			for id, record := range bc.blockVotes {
				if record.acceptable && bc.bitRecord.CompareBits(id) {
					newPreferred = &id
					break
				}
			}
		} else {
			// If this was just a bit flip and not a finalization, let's
			// first reset our current preference.
			if currentPreference.Compare(types.ID{}) != 0 {
				bc.blockVotes[currentPreference].Reset(false)
			}

			// Next let's selected a new preferred block if we can
			for id, record := range bc.blockVotes {
				// Select this record if:
				// - It was marked as acceptable
				// - It matches the current finalized bits
				// - It matches the active bit
				if record.acceptable && bc.bitRecord.CompareBits(id) &&
					(getBit(id, bc.bitRecord.activeBit) == 1) == bc.bitRecord.isOnePreferred() {

					newPreferred = &id
					break
				}
			}
		}
	} else {
		// All blocks are not preferred. Likely due to evenly distributed
		// votes across many blocks. Let's look through and see if any
		// are acceptable and reset that block as our new preference.
		if bc.GetPreference().Compare(types.ID{}) == 0 {
			for id, record := range bc.blockVotes {
				if record.acceptable && bc.bitRecord.CompareBits(id) {
					newPreferred = &id
					reselect = true
					break
				}
			}
		}
	}

	if newPreferred != nil {
		bc.blockVotes[*newPreferred].Reset(true)

		// When we finalize a bit we need to set the preference for
		// the active bit to that of our newly selected block.
		if result == ResultFinalized || reselect {
			bc.bitRecord.SetActiveBit(getBit(*newPreferred, bc.bitRecord.activeBit) == 1)
		}
	}

	return types.ID{}, false
}

// BitVoteRecord is responsible for tracking and finalizing bits.
// We start with the most significant bit (MSB) and attempt to finalize
// a 0 or 1 based on the MSB of the block ID votes. Since we're only
// making a binary choice, one should eventually finalize. We record
// the finalized bits and move on to trying to finalize the 2nd MSB.
type BitVoteRecord struct {
	activeBit     uint8
	finalizedBits types.ID

	votes      uint16
	consider   uint16
	confidence uint16
}

// RecordVote records a vote for active bit. If the bit finalizes
// we increment the active bit and reset the state to work on
// the next most significant bit.
func (vr *BitVoteRecord) RecordVote(voteID types.ID) Result {
	bit := getBit(voteID, vr.activeBit)
	if voteID.Compare(types.ID{}) == 0 {
		bit = 0x80
	}

	vr.votes = (vr.votes << 1) | boolToUint16(bit == 1)
	vr.consider = (vr.consider << 1) | boolToUint16(bit < 2)

	one := countBits16(vr.votes&vr.consider) > 12

	// The round is inconclusive
	if !one {
		zero := countBits16((-vr.votes-1)&vr.consider) > 12
		if !zero {
			return ResultNoChange
		}
	}

	// Vote is conclusive and agrees with our current state
	if vr.isOnePreferred() == one {
		vr.confidence += 2
		if vr.getConfidence() >= FinalizationScore {
			setBit(&vr.finalizedBits, vr.activeBit, vr.isOnePreferred())
			vr.activeBit++
			vr.votes = 0
			vr.confidence = 0
			vr.consider = 0
			return ResultFinalized
		}
		return ResultNoChange
	}

	// Vote is conclusive but does not agree with our current state
	vr.confidence = boolToUint16(one)
	return ResultFlipped
}

// SetActiveBit sets the value for the active bit
func (vr *BitVoteRecord) SetActiveBit(yes bool) {
	vr.confidence = boolToUint16(yes)
}

// CompareBits compares the ID to the bits we've already finalized
// and returns whether they match or not.
// It does not compare the active bit.
func (vr *BitVoteRecord) CompareBits(id types.ID) bool {
	if vr.activeBit == 0 {
		return true
	}
	return compareBits(vr.finalizedBits, id, vr.activeBit-1)
}

func (vr *BitVoteRecord) getConfidence() uint16 {
	return vr.confidence >> 1
}

func (vr *BitVoteRecord) isOnePreferred() bool {
	return (vr.confidence & 0x01) == 1
}

// BlockVoteRecord keeps track of votes for a single block ID
type BlockVoteRecord struct {
	acceptable bool

	votes      uint16
	consider   uint16
	confidence uint16
}

// RecordVote records the votes for a block ID and computes whether
// a block is finalized.
//
// Unlike bits, a block ID never finalizes as NO. It only remains in
// a NOT_PREFERRED state.
func (vr *BlockVoteRecord) RecordVote(vote byte) Result {
	vr.votes = (vr.votes << 1) | boolToUint16(vote == 1)
	vr.consider = (vr.consider << 1) | boolToUint16(vote < 2)

	yes := countBits16(vr.votes&vr.consider) > 12

	// The round is inconclusive
	if !yes {
		no := countBits16((-vr.votes-1)&vr.consider) > 12
		if !no {
			return ResultNoChange
		}
	}
	// Vote is conclusive and agrees with our current state
	if vr.isPreferred() == yes {
		if vr.isPreferred() {
			vr.confidence += 2
			if vr.getConfidence() >= FinalizationScore {
				return ResultFinalized
			}
		}
		return ResultNoChange
	}

	// Vote is conclusive but does not agree with our current state
	vr.confidence = boolToUint16(yes)
	return ResultFlipped
}

// Reset resets the state of the vote and sets the preference to
// the bool argument.
func (vr *BlockVoteRecord) Reset(yes bool) {
	vr.confidence = boolToUint16(yes)
	vr.votes = 0
	vr.consider = 0
}

// Status returns the current status of the block ID
func (vr *BlockVoteRecord) Status() (status Status) {
	finalized := vr.hasFinalized()
	preferred := vr.isPreferred()
	switch {
	case !finalized && preferred:
		status = StatusPreferred
	case !finalized && !preferred:
		status = StatusNotPreferred
	case finalized && preferred:
		status = StatusFinalized
	case finalized && !preferred:
		status = StatusRejected
	}
	return status
}

func (vr *BlockVoteRecord) isPreferred() bool {
	return (vr.confidence & 0x01) == 1
}

func (vr *BlockVoteRecord) hasFinalized() bool {
	return vr.getConfidence() >= FinalizationScore
}

func (vr *BlockVoteRecord) getConfidence() uint16 {
	return vr.confidence >> 1
}

func countBits8(i uint8) (count int) {
	for ; i > 0; i &= (i - 1) {
		count++
	}
	return count
}

func countBits16(i uint16) (count int) {
	for ; i > 0; i &= (i - 1) {
		count++
	}
	return count
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func boolToUint16(b bool) uint16 {
	return uint16(boolToUint8(b))
}

func getBit(id types.ID, pos uint8) uint8 {
	byteIndex := pos / 8
	bitIndex := pos % 8

	value := id[byteIndex]
	bit := (value >> (7 - bitIndex)) & 1
	return bit
}

func compareBits(a, b types.ID, x uint8) bool {
	// Determine the number of full bytes to compare
	fullBytes := (x + 1) / 8
	remainingBits := (x + 1) % 8

	// Compare full bytes
	for i := uint8(0); i < fullBytes; i++ {
		if a[i] != b[i] {
			return false
		}
	}

	// Check the remaining bits if there are any
	if remainingBits != 0 {
		mask := byte(255 << (8 - remainingBits)) // Create a mask for the remaining bits
		if (a[fullBytes] & mask) != (b[fullBytes] & mask) {
			return false
		}
	}

	return true
}

func setBit(arr *types.ID, bitIndex uint8, value bool) {
	byteIndex := bitIndex / 8     // Find the index of the byte
	bitPosition := 7 - bitIndex%8 // Find the position of the bit within the byte (MSB to LSB)

	if value {
		// Set the bit to 1
		arr[byteIndex] |= 1 << bitPosition
	} else {
		// Set the bit to 0
		arr[byteIndex] &^= 1 << bitPosition
	}
}
