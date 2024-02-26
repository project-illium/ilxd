// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"time"
)

// RequestRecord is a poll request for more votes
type RequestRecord struct {
	timestamp int64
	heights   []uint32
}

// NewRequestRecord creates a new RequestRecord
func NewRequestRecord(timestamp int64, heights []uint32) RequestRecord {
	return RequestRecord{timestamp, heights}
}

// GetTimestamp returns the timestamp that the request was created
func (r RequestRecord) GetTimestamp() int64 {
	return r.timestamp
}

// GetHeights returns the heights that were requested in this request.
func (r RequestRecord) GetHeights() []uint32 {
	return r.heights
}

// IsExpired returns true if the request has expired
func (r RequestRecord) IsExpired() bool {
	return time.Unix(r.timestamp, 0).Add(RequestTimeout).Before(time.Now())
}
