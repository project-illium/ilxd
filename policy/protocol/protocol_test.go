// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package protocol_test

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/policy/protocol"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPolicyProtocol(t *testing.T) {
	mn := mocknet.New()

	ds := mock.NewMapDatastore()

	host1, err := mn.GenPeer()
	assert.NoError(t, err)
	network1, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host1),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(ds),
		net.MaxMessageSize(repo.DefaultMaxMessageSize),
	}...)
	assert.NoError(t, err)

	host2, err := mn.GenPeer()
	assert.NoError(t, err)
	network2, err := net.NewNetwork(context.Background(), []net.Option{
		net.WithHost(host2),
		net.Params(&params.RegestParams),
		net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
			return nil
		}),
		net.MempoolValidator(func(transaction *transactions.Transaction) error {
			return nil
		}),
		net.Datastore(ds),
		net.MaxMessageSize(repo.DefaultMaxMessageSize),
	}...)
	assert.NoError(t, err)

	assert.NoError(t, mn.LinkAll())
	assert.NoError(t, mn.ConnectAllButSelf())

	p, err := policy.NewPolicy(nil, 0, 0, 0)
	assert.NoError(t, err)
	b, err := types.RandomSalt()
	assert.NoError(t, err)
	p.AddToTreasuryWhitelist(b)

	protocol.NewPolicyService(context.Background(), network1, &params.RegestParams, p)
	s2 := protocol.NewPolicyService(context.Background(), network2, &params.RegestParams, p)

	fpkb, err := s2.GetFeePerKb(network1.Host().ID())
	assert.NoError(t, err)
	assert.Equal(t, p.GetMinFeePerKilobyte(), fpkb)

	minStake, err := s2.GetMinStake(network1.Host().ID())
	assert.NoError(t, err)
	assert.Equal(t, p.GetMinStake(), minStake)

	limit, err := s2.GetBlocksizeSoftLimit(network1.Host().ID())
	assert.NoError(t, err)
	assert.Equal(t, p.GetBlocksizeSoftLimit(), limit)

	whitelist, err := s2.GetTreasuryWhitelist(network1.Host().ID())
	assert.NoError(t, err)
	assert.Len(t, whitelist, len(p.GetTreasuryWhitelist()))
	assert.Equal(t, whitelist[0], p.GetTreasuryWhitelist()[0])
}
