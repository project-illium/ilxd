// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockstore

import (
	"context"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeSerializeBlockLoc(t *testing.T) {
	location := BlockLocation{
		BlockFileNum: 1,
		FileOffset:   2,
		BlockLen:     3,
	}
	ser := SerializeBlockLoc(location)

	location2 := DeSerializeBlockLoc(ser)
	assert.Equal(t, location.BlockFileNum, location2.BlockFileNum)
	assert.Equal(t, location.FileOffset, location2.FileOffset)
	assert.Equal(t, location.BlockLen, location2.BlockLen)
}

func TestPutGetMany(t *testing.T) {
	blk := params.RegestParams.GenesisBlock

	ds, err := NewMockFlatFilestore(&params.RegestParams)
	assert.NoError(t, err)
	defer ds.Close()

	ser, err := blk.Serialize()
	assert.NoError(t, err)

	for i := 0; i < 100000; i++ {
		loc, err := ds.PutBlockData(0, ser)
		assert.NoError(t, err)

		blkData, err := ds.FetchBlockData(loc)
		assert.NoError(t, err)

		blk2 := new(blocks.Block)
		err = blk2.Deserialize(blkData)
		assert.NoError(t, err)

		assert.Equal(t, blk.ID(), blk2.ID())
	}
}

func TestFlatFilestore_DeleteBefore(t *testing.T) {
	blk := params.RegestParams.GenesisBlock

	ds, err := NewMockFlatFilestore(&params.RegestParams)
	assert.NoError(t, err)
	defer ds.Close()

	ser, err := blk.Serialize()
	assert.NoError(t, err)

	for i := uint32(0); i < 40000; i++ {
		_, err := ds.PutBlockData(i, ser)
		assert.NoError(t, err)
	}

	err = ds.DeleteBefore(22000)
	assert.NoError(t, err)

	assert.Len(t, ds.fileBlockHeights, 0)
}

func TestTransaction(t *testing.T) {
	blk := params.RegestParams.GenesisBlock

	ds, err := NewMockFlatFilestore(&params.RegestParams)
	assert.NoError(t, err)
	defer ds.Close()

	ser, err := blk.Serialize()
	assert.NoError(t, err)

	btx, err := ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	locs := make([]BlockLocation, 0, 40000)

	for i := uint32(0); i < 40000; i++ {
		loc, err := btx.PutBlockData(i, ser)
		assert.NoError(t, err)

		locs = append(locs, loc)
	}

	err = btx.Commit(context.Background())
	assert.NoError(t, err)

	btx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	for _, loc := range locs {
		_, err = btx.FetchBlockData(loc)
		assert.NoError(t, err)
	}

	btx.Discard(context.Background())

	btx, err = ds.NewTransaction(context.Background(), false)
	assert.NoError(t, err)

	locs = make([]BlockLocation, 0, 40000)

	for i := uint32(0); i < 40000; i++ {
		loc, err := btx.PutBlockData(i, ser)
		assert.NoError(t, err)

		locs = append(locs, loc)
	}

	btx.Discard(context.Background())

	for _, loc := range locs {
		_, err = btx.FetchBlockData(loc)
		assert.Error(t, err)
	}
}
