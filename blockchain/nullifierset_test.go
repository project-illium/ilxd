// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNullifierSet_Init(t *testing.T) {
	t.Run("empty db", func(t *testing.T) {
		ds := mock.NewMapDatastore()
		ns := NewNullifierSet(ds, 5000)

		prms := &params.RegestParams
		genesis := &blockNode{
			ds:      ds,
			blockID: prms.GenesisBlock.ID(),
			height:  0,
		}
		err := ns.Init(genesis)
		assert.NoError(t, err)

		status, err := dsFetchNullifierSetConsistencyStatus(ds)
		assert.NoError(t, err)
		assert.Equal(t, status, scsConsistent)
	})
}
