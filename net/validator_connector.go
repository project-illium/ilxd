// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"github.com/libp2p/go-libp2p/core/host"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/types"
	"sync"
	"time"
)

type validatorStat struct {
	queryAttempts uint64
	queryFailures uint64
}

// ValidatorConnector does two things.
// First, it strives to maintain active connections to all validators in the
// validator set.
// Second, it tracks the percentage of the weighted stake that we are connected to.
type ValidatorConnector struct {
	ownID               peer.ID
	connectedPercentage float64
	getValidatorFunc    func(validatorID peer.ID) (*blockchain.Validator, error)
	getValidatorsFunc   func() []*blockchain.Validator
	host                host.Host
	currentStats        map[peer.ID]*validatorStat
	lastEpochStats      map[peer.ID]*validatorStat

	mtx sync.RWMutex
}

// NewValidatorConnector returns a new ValidatorConnector
func NewValidatorConnector(host host.Host, ownID peer.ID,
	getValidatorFunc func(validatorID peer.ID) (*blockchain.Validator, error),
	getValidatorsFunc func() []*blockchain.Validator,
	blockchainSubscribeFunc func(cb blockchain.NotificationCallback)) *ValidatorConnector {

	vc := &ValidatorConnector{
		ownID:             ownID,
		getValidatorFunc:  getValidatorFunc,
		getValidatorsFunc: getValidatorsFunc,
		host:              host,
		currentStats:      make(map[peer.ID]*validatorStat),
		lastEpochStats:    make(map[peer.ID]*validatorStat),
		mtx:               sync.RWMutex{},
	}

	blockchainSubscribeFunc(vc.handleBlockchainNotification)

	host.Network().Notify(&inet.NotifyBundle{
		ConnectedF:    vc.handlePeerConnected,
		DisconnectedF: vc.handlePeerDisconnected,
	})

	go vc.run()
	return vc
}

// ConnectedStakePercentage returns the percentage of the weighted stake that
// we are connected to.
func (vc *ValidatorConnector) ConnectedStakePercentage() float64 {
	vc.mtx.RLock()
	defer vc.mtx.RUnlock()

	return vc.connectedPercentage
}

// RegisterDialSuccess registers a successful query for the purpose of
// tracking validator uptime and behavior.
func (vc *ValidatorConnector) RegisterDialSuccess(p peer.ID) {
	vc.mtx.Lock()
	defer vc.mtx.Unlock()

	val, ok := vc.currentStats[p]
	if ok {
		val.queryAttempts++
	} else {
		vc.currentStats[p] = &validatorStat{
			queryAttempts: 1,
		}
	}
}

// RegisterDialFailure registers a failed query for the purpose of
// tracking validator uptime and behavior.
func (vc *ValidatorConnector) RegisterDialFailure(p peer.ID) {
	vc.mtx.Lock()
	defer vc.mtx.Unlock()

	val, ok := vc.currentStats[p]
	if ok {
		val.queryAttempts++
		val.queryFailures++
	} else {
		vc.currentStats[p] = &validatorStat{
			queryAttempts: 1,
			queryFailures: 1,
		}
	}
}

// ValidatorDialSuccessRate returns the dial success rate for the validator
// for the prior epoch.
func (vc *ValidatorConnector) ValidatorDialSuccessRate(p peer.ID) float64 {
	vc.mtx.RLock()
	defer vc.mtx.RUnlock()

	stats, ok := vc.lastEpochStats[p]
	if !ok {
		return 100
	}
	if stats.queryAttempts == 0 {
		return 100
	}
	return float64(stats.queryAttempts-stats.queryFailures) / float64(stats.queryAttempts)
}

func (vc *ValidatorConnector) resetStats() {
	vc.mtx.Lock()
	defer vc.mtx.Unlock()

	vc.lastEpochStats = vc.currentStats
	vc.currentStats = make(map[peer.ID]*validatorStat)
}

func (vc *ValidatorConnector) run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		vc.update()
	}
}

func (vc *ValidatorConnector) update() {
	totalStake := types.Amount(0)
	connectedStake := types.Amount(0)
	if val, err := vc.getValidatorFunc(vc.ownID); err == nil {
		connectedStake = val.WeightedStake
	}

	for _, val := range vc.getValidatorsFunc() {
		totalStake += val.WeightedStake

		switch vc.host.Network().Connectedness(val.PeerID) {
		case inet.Connected:
			if val.PeerID != vc.ownID {
				connectedStake += val.WeightedStake
			}
		case inet.NotConnected, inet.CanConnect:
			if val.PeerID != vc.ownID {
				go vc.host.Connect(context.Background(), peer.AddrInfo{ID: val.PeerID})
			}
		}
	}

	vc.mtx.Lock()
	vc.connectedPercentage = float64(connectedStake) / float64(totalStake)
	vc.mtx.Unlock()
}

func (vc *ValidatorConnector) handleBlockchainNotification(ntf *blockchain.Notification) {
	if ntf.Type == blockchain.NTValidatorSetUpdate {
		vc.update()
	} else if ntf.Type == blockchain.NTNewEpoch {
		vc.resetStats()
	}
}

func (vc *ValidatorConnector) handlePeerConnected(_ inet.Network, conn inet.Conn) {
	_, err := vc.getValidatorFunc(conn.RemotePeer())
	if err == nil {
		vc.host.ConnManager().Protect(conn.RemotePeer(), ValidatorProtectionFlag)
		vc.update()
	}
}

func (vc *ValidatorConnector) handlePeerDisconnected(_ inet.Network, conn inet.Conn) {
	_, err := vc.getValidatorFunc(conn.RemotePeer())
	if err == nil {
		vc.host.ConnManager().Unprotect(conn.RemotePeer(), ValidatorProtectionFlag)
		vc.update()
	}
}
