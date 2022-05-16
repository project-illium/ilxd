// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

import (
	"context"
	"crypto/rand"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/project-illium/ilxd/net"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"sync"
	"testing"
)

func TestAvalancheEngine(t *testing.T) {
	mocknet := mocknet.New(context.Background())
	numNodes := 150
	numNoVotes := 0
	numAlwaysNoVotes := 0

	var (
		engines = make([]*AvalancheEngine, 0, numNodes)
	)

	for i := 0; i < numNodes; i++ {
		host, err := mocknet.GenPeer()
		if err != nil {
			t.Fatal(err)
		}
		network, err := net.NewNetwork(context.Background(), []net.Option{
			net.WithHost(host),
			net.Params(&params.RegestParams),
		}...)
		if err != nil {
			t.Fatal(err)
		}

		engine, err := NewAvalancheEngine(context.Background(), network)
		if err != nil {
			t.Fatal(err)
		}
		engines = append(engines, engine)
	}

	if err := mocknet.LinkAll(); err != nil {
		t.Fatal(err)
	}
	if err := mocknet.ConnectAllButSelf(); err != nil {
		t.Fatal(err)
	}

	for _, engine := range engines {
		engine.Start()
	}
	b := make([]byte, 32)
	rand.Read(b)
	chans := make([]chan Status, 0, numNodes)
	//start := time.Now()
	for i, engine := range engines {
		chans = append(chans, make(chan Status))
		if i < numAlwaysNoVotes {
			engine.alwaysNo = true
			continue
		}

		engine.NewBlock(types.NewID(b), i >= numNoVotes, chans[i])
	}

	var wg sync.WaitGroup
	wg.Add(numNodes - numAlwaysNoVotes)
	for i := 0; i < numNodes; i++ {
		if i < numAlwaysNoVotes {
			continue
		}
		go func(x int) {
			<-chans[x]
			//fmt.Printf("Node %d finished as %s\n", x, status)
			wg.Done()
		}(i)
	}
	wg.Wait()
	//fmt.Println(time.Since(start))
}
