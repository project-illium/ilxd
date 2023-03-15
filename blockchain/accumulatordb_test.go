// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"crypto/rand"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAccumulatorDB_Commit(t *testing.T) {
	// Create a new accumulator and fill it with entries.
	ds := mock.NewMapDatastore()
	adb := NewAccumulatorDB(ds)
	acc := NewAccumulator()
	start := time.Now()

	for i := 0; i < 1000; i++ {
		b := make([]byte, 32)
		rand.Read(b)
		acc.Insert(b, b[0] < 128)
	}

	// Commit entries and force a flush.
	err := adb.Commit(acc, 1, flushRequired)
	assert.NoError(t, err)

	// Check last flush height and consistency status are as expected.
	lastFlushHeight, err := dsFetchAccumulatorLastFlushHeight(ds)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, lastFlushHeight)

	cs, err := dsFetchAccumulatorConsistencyStatus(ds)
	assert.NoError(t, err)
	assert.EqualValues(t, scsConsistent, cs)

	// Check the accumulator on disk is as expected.
	dbAcc, err := dsFetchAccumulator(ds)
	assert.NoError(t, err)
	assert.True(t, accumulatorDeepEqual(acc, dbAcc))
	assert.True(t, adb.lastFlush.After(start))
	assert.True(t, adb.lastFlush.Before(time.Now()))

	// Add a new entry to the accumulator and commit but
	// using flushPeriodic. This should not flush as not
	// enough time has passed since the last flush.
	accOld := acc.Clone()
	acc.Insert([]byte{0x00}, false)
	err = adb.Commit(acc, 2, flushPeriodic)
	assert.NoError(t, err)

	// Confirm the last flush height and consistency status
	// has not changed.
	lastFlushHeight, err = dsFetchAccumulatorLastFlushHeight(ds)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, lastFlushHeight)

	cs, err = dsFetchAccumulatorConsistencyStatus(ds)
	assert.NoError(t, err)
	assert.EqualValues(t, scsConsistent, cs)

	// Make sure the accumulator on disk remains unchanged.
	dbAcc, err = dsFetchAccumulator(ds)
	assert.NoError(t, err)
	assert.True(t, accumulatorDeepEqual(accOld, dbAcc))

	// Make sure the accumulator in memory is as expected.
	assert.True(t, accumulatorDeepEqual(acc, adb.Accumulator()))
}

func TestAccumulatorDB_Init(t *testing.T) {
	// Create a new accumulator and fill it with entries.
	ds := mock.NewMapDatastore()
	adb := NewAccumulatorDB(ds)
	acc := NewAccumulator()

	for i := 0; i < 1000; i++ {
		b := make([]byte, 32)
		rand.Read(b)
		acc.Insert(b, b[0] < 128)
	}

	// Commit entries and force a flush.
	err := adb.Commit(acc, 1, flushRequired)
	assert.NoError(t, err)

	// Recreate the accumulator db.
	adb = NewAccumulatorDB(ds)
	// Init with the db height the same as the tip.
	err = adb.Init(&blockNode{
		ds:     ds,
		height: 1,
	})
	assert.NoError(t, err)

	// Accumulator should be the same.
	assert.True(t, accumulatorDeepEqual(acc, adb.Accumulator()))

	// Next we're going to test initializing from a db that
	// is not synced to the tip.
	index, err := mockBlockIndex(ds, 10)
	assert.NoError(t, err)

	err = adb.Init(index.Tip())
	assert.NoError(t, err)

	// Confirm the last flush height and consistency status
	// has not changed.
	lastFlushHeight, err := dsFetchAccumulatorLastFlushHeight(ds)
	assert.NoError(t, err)
	assert.EqualValues(t, 9, lastFlushHeight)

	cs, err := dsFetchAccumulatorConsistencyStatus(ds)
	assert.NoError(t, err)
	assert.EqualValues(t, scsConsistent, cs)

	// Accumulator should have the original amount of elements
	// plus 8 mock blocks worth.
	assert.EqualValues(t, 80+1000, adb.Accumulator().NumElements())
}
