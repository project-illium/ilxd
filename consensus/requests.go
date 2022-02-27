// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/project-illium/ilxd/models"
	"time"
)

// RequestRecord is a poll request for more votes
type RequestRecord struct {
	timestamp int64
	invs      []models.ID
}

// NewRequestRecord creates a new RequestRecord
func NewRequestRecord(timestamp int64, invs []models.ID) RequestRecord {
	return RequestRecord{timestamp, invs}
}

// GetTimestamp returns the timestamp that the request was created
func (r RequestRecord) GetTimestamp() int64 {
	return r.timestamp
}

// GetInvs returns the poll Invs for the request
func (r RequestRecord) GetInvs() map[models.ID]bool {
	m := make(map[models.ID]bool)
	for _, inv := range r.invs {
		m[inv] = true
	}
	return m
}

// IsExpired returns true if the request has expired
func (r RequestRecord) IsExpired() bool {
	return time.Unix(r.timestamp, 0).Add(AvalancheRequestTimeout).Before(time.Now())
}
