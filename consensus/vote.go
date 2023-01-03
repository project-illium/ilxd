// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"fmt"
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

// VoteRecord keeps track of a series of votes for a target
type VoteRecord struct {
	blockID          types.ID
	votes            uint16
	consider         uint16
	confidence       uint16
	inflightRequests uint8
	timestamp        time.Time
	totalVotes       int
}

// NewVoteRecord instantiates a new base record for voting on a target
// `accepted` indicates whether or not the initial state should be preferred
func NewVoteRecord(blockID types.ID, preferred bool) *VoteRecord {
	return &VoteRecord{blockID: blockID, confidence: boolToUint16(preferred), timestamp: time.Now()}
}

// isPreferred returns whether or not the voted state is preferred or not
func (vr VoteRecord) isPreferred() bool {
	return (vr.confidence & 0x01) == 1
}

// getConfidence returns the confidence in the current state's finalization
func (vr VoteRecord) getConfidence() uint16 {
	return vr.confidence >> 1
}

// hasFinalized returns whether or not the record has finalized a state
func (vr VoteRecord) hasFinalized() bool {
	return vr.getConfidence() >= AvalancheFinalizationScore
}

// regsiterVote adds a new vote for an item and update confidence accordingly.
// Returns true if the acceptance or finalization state changed.
func (vr *VoteRecord) regsiterVote(vote uint8) bool {
	vr.totalVotes++
	vr.votes = (vr.votes << 1) | boolToUint16(vote == 1)
	vr.consider = (vr.consider << 1) | boolToUint16(vote < 2)

	yes := countBits16(vr.votes&vr.consider) > 12

	// The round is inconclusive
	if !yes {
		no := countBits16((-vr.votes-1)&vr.consider) > 12
		if !no {
			return false
		}
	}

	// Vote is conclusive and agrees with our current state
	if vr.isPreferred() == yes {
		vr.confidence += 2
		return vr.getConfidence() == AvalancheFinalizationScore
	}

	// Vote is conclusive but does not agree with our current state
	vr.confidence = boolToUint16(yes)

	return true
}

func (vr *VoteRecord) status() (status Status) {
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

func (vr *VoteRecord) printState() {
	fmt.Printf("Votes: %016b\n", vr.votes)
	fmt.Printf("Consider: %016b\n", vr.consider)
	fmt.Printf("Confidence: %016b\n", vr.confidence)
	fmt.Printf("Total Votes: %d\n", vr.totalVotes)
	fmt.Println()
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
