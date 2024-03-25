// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package params

import (
	"encoding/json"
	"github.com/project-illium/ilxd/types"
)

// Checkpoint represents a known good block in the blockchain
type Checkpoint struct {
	BlockID types.ID
	Height  uint32
}

type jsonCheckpoint struct {
	BlockID types.HexEncodable `json:"blockID"`
	Height  uint32             `json:"height"`
}

// MarshalJSON implements the json marshal interface
func (c *Checkpoint) MarshalJSON() ([]byte, error) {
	s := &jsonCheckpoint{
		BlockID: c.BlockID.Bytes(),
		Height:  c.Height,
	}
	return json.Marshal(s)
}

// UnmarshalJSON implements the json unmarshal interface
func (c *Checkpoint) UnmarshalJSON(data []byte) error {
	newCheckpoint := &jsonCheckpoint{}
	if err := json.Unmarshal(data, newCheckpoint); err != nil {
		return err
	}

	*c = Checkpoint{
		BlockID: types.NewID(newCheckpoint.BlockID),
		Height:  newCheckpoint.Height,
	}
	return nil
}

// CheckpointSorter implements sort.Interface to allow a slice of checkpoints
// ids to be sorted by height.
type CheckpointSorter []Checkpoint

// Len returns the number of checkpoints in the slice.  It is part of the
// sort.Interface implementation.
func (s CheckpointSorter) Len() int {
	return len(s)
}

// Swap swaps the checkpoints at the passed indices.  It is part of the
// sort.Interface implementation.
func (s CheckpointSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the checkpoint with index i should sort before the
// checkpoint with index j.  It is part of the sort.Interface implementation.
func (s CheckpointSorter) Less(i, j int) bool {
	return s[i].Height < s[j].Height
}
