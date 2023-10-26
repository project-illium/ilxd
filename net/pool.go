// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"sync"
	"time"
)

type messageSenderPool struct {
	pool       []*peerMessageSender
	mu         sync.Mutex
	gcInterval time.Duration
	lastUsed   map[*peerMessageSender]time.Time
	new        func() *peerMessageSender
	done       chan struct{}
}

func newMessageSenderPool(gcInterval time.Duration, new func() *peerMessageSender) *messageSenderPool {
	p := &messageSenderPool{
		gcInterval: gcInterval,
		lastUsed:   make(map[*peerMessageSender]time.Time),
		new:        new,
		done:       make(chan struct{}),
	}
	go p.garbageCollector()
	return p
}

func (p *messageSenderPool) Get() *peerMessageSender {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pool) == 0 {
		sender := p.new()
		p.lastUsed[sender] = time.Now()
		return sender
	}
	sender := p.pool[len(p.pool)-1]
	p.pool = p.pool[:len(p.pool)-1]
	p.lastUsed[sender] = time.Now()
	return sender
}

func (p *messageSenderPool) Put(sender *peerMessageSender) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = append(p.pool, sender)
	p.lastUsed[sender] = time.Now()
}

func (p *messageSenderPool) Close() {
	close(p.done)
}

func (p *messageSenderPool) garbageCollector() {
	ticker := time.NewTicker(p.gcInterval)
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.runGC()
		}
	}
}

func (p *messageSenderPool) runGC() {
	p.mu.Lock()
	for sender, lastUsed := range p.lastUsed {
		if time.Since(lastUsed) > p.gcInterval && len(p.pool) > 1 {
			// Find and remove sender from pool
			for i, s := range p.pool {
				if s == sender {
					p.pool = append(p.pool[:i], p.pool[i+1:]...)
					break
				}
			}
			// Cleanup and remove from lastUsed map
			sender.teardown()
			delete(p.lastUsed, sender)
		}
	}
	p.mu.Unlock()
}
