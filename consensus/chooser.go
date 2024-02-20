// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/cenkalti/backoff/v4"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"time"
)

type backoffTime struct {
	backoffUntil time.Time
	eb           *backoff.ExponentialBackOff
}

// BackoffChooser wraps the WeightedRandomChooser with a map
// that tracks exponential backoffs for peer dials.
type BackoffChooser struct {
	peerMap map[peer.ID]*backoffTime
	chooser blockchain.WeightedChooser
	valconn ValidatorSetConnection
}

// NewBackoffChooser returns a new initialized BackoffChooser
func NewBackoffChooser(chooser blockchain.WeightedChooser, valconn ValidatorSetConnection) *BackoffChooser {
	return &BackoffChooser{
		peerMap: make(map[peer.ID]*backoffTime),
		chooser: chooser,
		valconn: valconn,
	}
}

// WeightedRandomValidator returns a weighted random validator.
// If the selected validator is undergoing a backoff weight time
// then "" will be returned.
func (b *BackoffChooser) WeightedRandomValidator() peer.ID {
	peer := b.chooser.WeightedRandomValidator()
	if bot, ok := b.peerMap[peer]; ok {
		if time.Now().After(bot.backoffUntil) {
			return peer

		}
		// We'll count this as a dial failure for purposes of tracking
		// peer uptime.
		b.valconn.RegisterDialFailure(peer)
		return ""
	}
	return peer
}

// RegisterDialFailure increases the exponential backoff time for
// the given peer.
func (b *BackoffChooser) RegisterDialFailure(p peer.ID) {
	b.valconn.RegisterDialFailure(p)
	bot, ok := b.peerMap[p]
	if ok {
		t := bot.eb.NextBackOff()
		b.peerMap[p].backoffUntil = time.Now().Add(t)
		return
	}
	eb := &backoff.ExponentialBackOff{
		InitialInterval:     backoff.DefaultInitialInterval,
		RandomizationFactor: 0,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         backoff.DefaultMaxInterval,
		MaxElapsedTime:      0,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	eb.Reset()
	b.peerMap[p] = &backoffTime{
		backoffUntil: time.Now().Add(eb.NextBackOff()),
		eb:           eb,
	}
	log.WithCaller(true).Trace("Adding dial backoff", log.Args("peer", p))
}

// RegisterDialSuccess deletes the exponential backoff for the
// given peer.
func (b *BackoffChooser) RegisterDialSuccess(p peer.ID) {
	b.valconn.RegisterDialSuccess(p)
	_, ok := b.peerMap[p]
	if ok {
		log.WithCaller(true).Trace("Removing dial backoff", log.Args("peer", p))
		delete(b.peerMap, p)
	}
}
