// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/project-illium/ilxd/types"
	"time"
)

// Status is the status of consensus on a particular target
type Status int

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

type Result uint8

const (
	ResultNoChange = iota
	ResultFlipped
	ResultFinalized
)

type BlockChoice struct {
	height           uint32
	bitRecord        *BitVoteRecord
	blockVotes       map[types.ID]*BlockVoteRecord
	inflightRequests int
	timestamp        time.Time
	totalVotes       int
}

func NewBlockChoice(height uint32) *BlockChoice {
	return &BlockChoice{
		height:     height,
		bitRecord:  &BitVoteRecord{},
		blockVotes: make(map[types.ID]*BlockVoteRecord),
		timestamp:  time.Now(),
	}
}

func (bc *BlockChoice) GetPreference() types.ID {
	for id, rec := range bc.blockVotes {
		if rec.isPreferred() {
			return id.Clone()
		}
	}
	return types.ID{}
}

func (bc *BlockChoice) HasFinalized() bool {
	for _, rec := range bc.blockVotes {
		if rec.Status() == StatusFinalized {
			return true
		}
	}
	return false
}

func (bc *BlockChoice) VotesNeededToFinalize() int {
	max := AvalancheMaxInflightPoll
	for _, rec := range bc.blockVotes {
		confidence := rec.getConfidence()
		if AvalancheMaxInflightPoll-int(confidence) < max {
			max = AvalancheFinalizationScore - int(confidence)
		}
	}
	return max
}

func (bc *BlockChoice) HasBlock(id types.ID) bool {
	_, ok := bc.blockVotes[id]
	return ok
}

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
			bc.bitRecord.SetBit(getBit(blockID, bc.bitRecord.activeBit) == 1)
		}
	}

	bc.blockVotes[blockID] = &BlockVoteRecord{
		acceptable: isAcceptable,
		confidence: boolToUint16(preferred),
	}
}

func (bc *BlockChoice) RecordVote(voteID types.ID) (types.ID, bool) {
	bc.totalVotes++

	// Set the vote to unknown if the voteID is all zeros.
	v1, v2 := byte(0x01), byte(0x00)
	if voteID.Compare(types.ID{}) == 0 {
		v1, v2 = 0x80, 0x80
	}

	// Record the new block vote and check for finalization
	record, ok := bc.blockVotes[voteID]
	if ok {
		if record.RecordVote(v1) == ResultFinalized {
			return voteID, true
		}
	}

	// Iterate over all other blocks and record a no vote
	for id, record := range bc.blockVotes {
		if id != voteID {
			if record.RecordVote(v2) == ResultFinalized {
				return id, true
			}
		}
	}

	// Record the bit vote based on the voteID and check to
	// see if our bit preference either flipped or finalized.
	if result := bc.bitRecord.RecordVote(voteID); result != ResultNoChange {
		var currentPreference types.ID
		for id, record := range bc.blockVotes {
			if record.isPreferred() {
				currentPreference = id
				break
			}
		}
		var newPreferred *types.ID
		if result == ResultFinalized {
			// The current preference matches the newly finalized bits, so we
			// don't need to do anything.
			if currentPreference.Compare(types.ID{}) != 0 && bc.bitRecord.CompareBits(currentPreference) {
				return types.ID{}, false
			}

			// Loop through the existing records and see if we can't find one that matches
			// the newly finalized bits
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
				// - The active bit matches
				if record.acceptable && bc.bitRecord.CompareBits(id) &&
					(getBit(id, bc.bitRecord.activeBit) == 1) == bc.bitRecord.isOnePreferred() {

					newPreferred = &id
					break
				}
			}
		}

		if newPreferred != nil {
			bc.blockVotes[*newPreferred].Reset(true)

			// When we finalize a bit we need to set the preference for
			// the active bit to that of our newly selected block.
			if result == ResultFinalized {
				bc.bitRecord.SetBit(getBit(*newPreferred, bc.bitRecord.activeBit) == 1)
			}
		}
	}

	return types.ID{}, false
}

type BitVoteRecord struct {
	activeBit     uint8
	finalizedBits types.ID

	votes      uint16
	consider   uint16
	confidence uint16
}

func (vr *BitVoteRecord) RecordVote(voteID types.ID) Result {
	bit := getBit(voteID, vr.activeBit)
	if voteID.Compare(types.ID{}) == 0 {
		bit = 0x80
	}

	vr.votes = (vr.votes << 1) | boolToUint16(bit == 1)
	vr.consider = (vr.consider << 1) | boolToUint16(bit < 2)

	one := countBits16(vr.votes&vr.consider) > 11

	// The round is inconclusive
	if !one {
		zero := countBits16((-vr.votes-1)&vr.consider) > 11
		if !zero {
			return ResultNoChange
		}
	}

	// Vote is conclusive and agrees with our current state
	if vr.isOnePreferred() == one {
		vr.confidence += 2
		if vr.getConfidence() >= AvalancheFinalizationScore {
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

func (vr *BitVoteRecord) SetBit(yes bool) {
	vr.confidence = boolToUint16(yes)
}

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

type BlockVoteRecord struct {
	acceptable bool

	votes      uint16
	consider   uint16
	confidence uint16
}

func (vr *BlockVoteRecord) RecordVote(vote byte) Result {
	vr.votes = (vr.votes << 1) | boolToUint16(vote == 1)
	vr.consider = (vr.consider << 1) | boolToUint16(vote < 2)

	yes := countBits16(vr.votes&vr.consider) > 11

	// The round is inconclusive
	if !yes {
		no := countBits16((-vr.votes-1)&vr.consider) > 11
		if !no {
			return ResultNoChange
		}
	}
	// Vote is conclusive and agrees with our current state
	if vr.isPreferred() == yes {
		if vr.isPreferred() {
			vr.confidence += 2
			if vr.getConfidence() >= AvalancheFinalizationScore {
				return ResultFinalized
			}
		}
		return ResultNoChange
	}

	// Vote is conclusive but does not agree with our current state
	vr.confidence = boolToUint16(yes)
	return ResultFlipped
}

func (vr *BlockVoteRecord) Reset(yes bool) {
	vr.confidence = boolToUint16(yes)
	vr.votes = 0
	vr.consider = 0
}

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
	return vr.getConfidence() >= AvalancheFinalizationScore
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
