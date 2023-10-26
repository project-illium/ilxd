// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMessageSenderPool(t *testing.T) {
	// Set a short GC interval for the purpose of this test
	gcInterval := time.Millisecond * 100
	pool := newMessageSenderPool(gcInterval, func() *peerMessageSender {
		return &peerMessageSender{
			lk: NewCtxMutex(),
		}
	})

	// Test getting a new peerMessageSender
	sender1 := pool.Get()
	assert.NotNil(t, sender1)

	// Test returning the peerMessageSender to the pool
	pool.Put(sender1)

	// Test getting the same peerMessageSender back from the pool
	sender2 := pool.Get()
	assert.True(t, sender1 == sender2)

	// This should be a new peerMessageSender
	sender3 := pool.Get()

	// Return the peerMessageSender to the pool and wait for the GC to run
	pool.Put(sender2)
	pool.Put(sender3)
	time.Sleep(gcInterval * 2)

	// Acquire a new peerMessageSender and check that the old one was cleaned up
	sender4 := pool.Get()
	assert.True(t, sender1 != sender4)
	assert.True(t, sender2 != sender4)
}
