// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/repo/mock"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	mrand "math/rand"
	"sync"
	"testing"
	"time"
)

// MockChooser is a mock WeightedChooser for testing.
type MockChooser struct {
	network *net.Network
}

// WeightedRandomValidator returns a validator weighted by their current stake.
func (m *MockChooser) WeightedRandomValidator() peer.ID {
	peers := m.network.Host().Network().Peers()
	l := len(peers)
	if l == 0 {
		return ""
	}
	i := mrand.Intn(l)
	return peers[i]
}

func TestAvalancheEngine(t *testing.T) {
	numNodes := 100
	numNoVotes := 45
	numAlwaysNoVotes := 0
	mrand.Seed(time.Now().Unix())

	nFinalized := 0
	for i := 0; i < 10; i++ {
		finalized, err := runTest(numNodes, numNoVotes, numAlwaysNoVotes)
		if err != nil {
			t.Fatal(err)
		}
		if finalized {
			nFinalized++
		}
	}

	fmt.Println(nFinalized)
}

func runTest(numNodes int, numNoVotes int, numAlwaysNoVotes int) (bool, error) {
	var (
		mocknet = mocknet.New()
		engines = make([]*ConsensusEngine, 0, numNodes)
	)

	defer mocknet.Close()
	for i := 0; i < numNodes; i++ {
		host, err := mocknet.GenPeer()
		if err != nil {
			return false, err
		}
		network, err := net.NewNetwork(context.Background(), []net.Option{
			net.WithHost(host),
			net.Params(&params.RegestParams),
			net.BlockValidator(func(*blocks.XThinnerBlock, peer.ID) error {
				return nil
			}),
			net.MempoolValidator(func(transaction *transactions.Transaction) error {
				return nil
			}),
			net.Datastore(mock.NewMapDatastore()),
		}...)
		if err != nil {
			return false, err
		}

		engine, err := NewConsensusEngine(context.Background(),
			Params(&params.RegestParams),
			Network(network),
			Chooser(&MockChooser{network: network}),
			HasBlock(func(id types.ID) bool { return false }),
			RequestBlock(func(id types.ID, id2 peer.ID) {}),
		)
		if err != nil {
			return false, err
		}
		if i == 0 {
			//engine.printState = true
		}
		engines = append(engines, engine)
	}

	if err := mocknet.LinkAll(); err != nil {
		return false, err
	}
	if err := mocknet.ConnectAllButSelf(); err != nil {
		return false, err
	}

	for i, engine := range engines {
		if i < numAlwaysNoVotes {
			engine.alwaysNo = true
		}
	}
	defer func() {
		for _, eng := range engines {
			eng.Close()
		}
	}()
	b := &blocks.BlockHeader{
		Height: 3,
	}
	chans := make([]chan Status, 0, numNodes-numAlwaysNoVotes)
	start := time.Now()
	for o, engine := range engines {
		r := mrand.Intn(numNodes)
		var c chan Status
		if !engine.alwaysNo && !engine.flipFlopper {
			c = make(chan Status)
			chans = append(chans, c)
			time.AfterFunc(time.Second*120, func() {
				if engine.voteRecords[b.ID()].status() == StatusNotPreferred {
					c <- StatusRejected
				}
			})
		}
		engine.NewBlock(b, r >= numNoVotes, c)
		if o == 0 {
			go func() {
				for {
					_, ok := engine.voteRecords[b.ID()]
					if ok {
						//engine.voteRecords[b.ID()].print = true
						return
					}
				}
			}()

		}
	}

	finalized := 0
	rejected := 0
	var wg sync.WaitGroup
	wg.Add(numNodes - numAlwaysNoVotes)
	for i := 0; i < numNodes-numAlwaysNoVotes; i++ {
		go func(x int, w *sync.WaitGroup) {
			status := <-chans[x]
			if status == StatusFinalized {
				finalized++
			} else {
				rejected++
			}
			//fmt.Printf("Node %d finished as %s\n", x, status)
			w.Done()
		}(i, &wg)
	}
	wg.Wait()
	fmt.Println(time.Since(start), finalized, rejected)
	return finalized > 0, nil
}
