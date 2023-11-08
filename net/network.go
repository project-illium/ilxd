// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"errors"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	coreconmgr "github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"time"
)

const (
	BlockTopic              = "blocks"
	TransactionsTopic       = "transactions"
	RelayKey                = "/ilx/relaypeers"
	ValidatorProtectionFlag = "validator"
)

type Network struct {
	host        host.Host
	connManager coreconmgr.ConnManager
	connGater   *ConnectionGater
	dht         *dht.IpfsDHT
	pubsub      *pubsub.PubSub
	txTopic     *pubsub.Topic
	blockTopic  *pubsub.Topic
	pstoreds    *Peerstoreds
	txSub       *pubsub.Subscription
	blkSub      *pubsub.Subscription
}

func NewNetwork(ctx context.Context, opts ...Option) (*Network, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	var (
		self peer.ID
		err  error
	)
	if cfg.host == nil {
		self, err = peer.IDFromPrivateKey(cfg.privateKey)
		if err != nil {
			return nil, err
		}
	} else {
		self = cfg.host.ID()
	}

	seedAddrs := make([]peer.AddrInfo, 0, len(cfg.seedAddrs))
	for _, addr := range cfg.seedAddrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("%w: malformatted seed peer", ErrNetworkConfig)
		}

		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return nil, err
		}
		if pi.ID == self {
			cfg.forceServerMode = true
			continue
		}
		seedAddrs = append(seedAddrs, *pi)
	}

	var (
		kdht   *dht.IpfsDHT
		pstore peerstore.Peerstore
		cmgr   coreconmgr.ConnManager
	)
	cmgr, err = connmgr.NewConnManager(
		100, // Lowwater
		400, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, err
	}

	if cfg.host == nil {
		pstore, err = pstoremem.NewPeerstore()
		if err != nil {
			return nil, err
		}
	} else {
		pstore = cfg.host.Peerstore()
	}

	pstoreds := NewPeerstoreds(cfg.datastore, pstore)
	addrInfos, err := pstoreds.AddrInfos()
	if err != nil {
		return nil, err
	}
loop:
	for i, ai := range addrInfos {
		for _, s := range seedAddrs {
			if ai.ID == s.ID || ai.ID == self {
				continue loop
			}
		}
		seedAddrs = append(seedAddrs, ai)
		if i > 50 {
			break
		}
	}

	conngater, err := NewConnectionGater(cfg.datastore, pstore, cfg.banDuration, cfg.maxBanscore)
	if err != nil {
		return nil, err
	}

	mode := dht.ModeAuto
	if cfg.forceServerMode {
		mode = dht.ModeServer
	}

	dhtOpts := []dht.Option{
		dht.DisableValues(),
		dht.ProtocolPrefix(cfg.params.ProtocolPrefix),
		dht.BootstrapPeers(seedAddrs...),
		dht.Mode(mode),
	}

	peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		h, err := mh.Sum([]byte(RelayKey), mh.SHA2_256, -1)
		if err != nil {
			return nil
		}

		id := cid.NewCidV1(cid.Raw, h)
		return kdht.FindProvidersAsync(ctx, id, numPeers)
	}

	// Start with the default scaling limits.
	scalingLimits := rcmgr.DefaultLimits

	// Add limits around included libp2p protocols
	libp2p.SetDefaultServiceLimits(&scalingLimits)

	// Turn the scaling limits into a concrete set of limits using `.AutoScale`. This
	// scales the limits proportional to your system memory.
	scaledDefaultLimits := scalingLimits.AutoScale()

	rCfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			Streams:         rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			StreamsOutbound: rcmgr.Unlimited,
			Conns:           rcmgr.Unlimited,
			ConnsInbound:    rcmgr.Unlimited,
			ConnsOutbound:   rcmgr.Unlimited,
		},
		PeerDefault: rcmgr.ResourceLimits{
			Streams:         512,
			StreamsInbound:  256,
			StreamsOutbound: 256,
			Conns:           8,
			ConnsInbound:    8,
			ConnsOutbound:   8,
		},
		ProtocolDefault: rcmgr.ResourceLimits{
			Streams:         rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			StreamsOutbound: rcmgr.Unlimited,
			Conns:           rcmgr.Unlimited,
			ConnsInbound:    rcmgr.Unlimited,
			ConnsOutbound:   rcmgr.Unlimited,
			Memory:          rcmgr.Unlimited64,
		},
		ProtocolPeerDefault: rcmgr.ResourceLimits{
			Streams:         rcmgr.Unlimited,
			StreamsInbound:  rcmgr.Unlimited,
			StreamsOutbound: rcmgr.Unlimited,
			Conns:           rcmgr.Unlimited,
			ConnsInbound:    rcmgr.Unlimited,
			ConnsOutbound:   rcmgr.Unlimited,
			Memory:          rcmgr.Unlimited64,
		},
	}

	// Create our limits by using our cfg and replacing the default values with values from `scaledDefaultLimits`
	limits := rCfg.Build(scaledDefaultLimits)

	// The resource manager expects a limiter, se we create one from our limits.
	limiter := rcmgr.NewFixedLimiter(limits)

	// Metrics are enabled by default. If you want to disable metrics, use the
	// WithMetricsDisabled option
	// Initialize the resource manager
	rm, err := rcmgr.NewResourceManager(limiter, rcmgr.WithMetricsDisabled())
	if err != nil {
		panic(err)
	}

	hostOpts := libp2p.ChainOptions(
		// Use the keypair we generated
		libp2p.Identity(cfg.privateKey),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(cfg.listenAddrs...),
		// Noise and TLS
		libp2p.DefaultSecurity,

		// QUIC and TCP
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),

		libp2p.DefaultMuxers,

		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(cmgr),

		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			kdht, err = dht.New(ctx, h, dhtOpts...)
			return kdht, err
		}),

		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		libp2p.EnableAutoRelayWithPeerSource(peerSource),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),

		libp2p.EnableRelay(),

		libp2p.EnableHolePunching(),

		libp2p.UserAgent(cfg.userAgent),

		libp2p.Ping(true),

		libp2p.Peerstore(pstore),

		libp2p.ConnectionGater(conngater),

		libp2p.ResourceManager(rm),
	)

	if !cfg.disableNatPortMap {
		hostOpts = libp2p.ChainOptions(libp2p.NATPortMap(), hostOpts)
	}
	if cfg.forceServerMode {
		libp2p.ForceReachabilityPublic()
	}

	var host host.Host
	if cfg.host != nil {
		host = cfg.host
		cmgr = host.ConnManager()
		kdht, err = dht.New(ctx, cfg.host, dhtOpts...)
		if err != nil {
			return nil, err
		}
	} else {
		host, err = libp2p.New(hostOpts)
		if err != nil {
			return nil, err
		}
	}

	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(
		ctx,
		host,
		pubsub.WithNoAuthor(),
		pubsub.WithDiscovery(discovery.NewRoutingDiscovery(kdht)),
		pubsub.WithMaxMessageSize(cfg.maxMessageSize),
		pubsub.WithMessageIdFn(func(pmsg *pb.Message) string {
			h := hash.HashFunc(pmsg.Data)
			return string(h[:])
		}),
		pubsub.WithGossipSubProtocols([]protocol.ID{cfg.params.ProtocolPrefix + pubsub.GossipSubID_v11}, func(feature pubsub.GossipSubFeature, id protocol.ID) bool {
			if feature == pubsub.GossipSubFeatureMesh || feature == pubsub.GossipSubFeaturePX {
				return true
			}
			return false
		}),
	)
	if err != nil {
		return nil, err
	}

	err = ps.RegisterTopicValidator(TransactionsTopic, pubsub.ValidatorEx(func(ctx context.Context, p peer.ID, m *pubsub.Message) pubsub.ValidationResult {
		tx := &transactions.Transaction{}
		if err := tx.Deserialize(m.Data); err != nil {
			return pubsub.ValidationReject
		}
		err := cfg.acceptToMempool(tx)
		switch e := err.(type) {
		case mempool.PolicyError:
			// Policy errors do no penalize peer
			log.Debugf("Mempool reject tx %s. Policy error: %s:%s", tx.ID(), e.ErrorCode, e.Description)
			return pubsub.ValidationIgnore
		case blockchain.RuleError:
			// Rule errors do
			log.Debugf("Mempool reject tx %s. Rule error: %s:%s", tx.ID(), e.ErrorCode, e.Description)
			return pubsub.ValidationReject
		case blockchain.NotCurrentError:
			return pubsub.ValidationIgnore
		case nil:
			return pubsub.ValidationAccept
		default:
			log.Debugf("Mempool reject tx %s. Unknown error: %s", tx.ID(), err)
			return pubsub.ValidationIgnore
		}
	}))
	if err != nil {
		return nil, err
	}

	// For blocks we will wait for the full block to be recovered from the compact block
	// so that we can validate it before returning here.
	err = ps.RegisterTopicValidator(BlockTopic, pubsub.ValidatorEx(func(ctx context.Context, p peer.ID, m *pubsub.Message) pubsub.ValidationResult {
		blk := &blocks.XThinnerBlock{}
		if err := blk.Deserialize(m.Data); err != nil {
			log.Errorf("[PUBSUB] xthinner deserialize error: %s", err)
			return pubsub.ValidationReject
		}
		log.Debugf("[PUBSUB] new incoming block: %s", blk.ID())
		err := cfg.validateBlock(blk, p)
		switch e := err.(type) {
		case blockchain.OrphanBlockError:
			// Orphans we won't relay (yet) but won't penalize them either.
			log.Debugf("Recieved orphan block: %s", blk.ID())
			return pubsub.ValidationIgnore
		case blockchain.RuleError:
			// Rule errors do
			log.Debugf("Block %s rule error: %s:%s", blk.ID(), e.ErrorCode, e.Description)
			return pubsub.ValidationReject
		case blockchain.NotCurrentError:
			return pubsub.ValidationIgnore
		case nil:
			return pubsub.ValidationAccept
		default:
			log.Debugf("Block reject %s. Unknown error: %s", blk.ID(), err)
			return pubsub.ValidationIgnore
		}
	}))
	if err != nil {
		return nil, err
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	protocolUpdatedSub, err := host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated))
	if err != nil {
		return nil, err
	}
	go func(sub event.Subscription) {
		for evt := range sub.Out() {
			event, ok := evt.(event.EvtPeerProtocolsUpdated)
			if !ok {
				return
			}

			var updated bool
			for _, proto := range event.Added {
				if proto == cfg.params.ProtocolPrefix+pubsub.GossipSubID_v11 {
					updated = true
					break
				}
			}

			if updated {
				for _, c := range host.Network().ConnsToPeer(event.Peer) {
					(*pubsub.PubSubNotif)(ps).Connected(host.Network(), c)
				}
			}
		}
	}(protocolUpdatedSub)

	txTopic, err := ps.Join(TransactionsTopic)
	if err != nil {
		return nil, err
	}

	blockTopic, err := ps.Join(BlockTopic)
	if err != nil {
		return nil, err
	}

	txSub, err := txTopic.Subscribe()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			_, err := txSub.Next(context.Background())
			if errors.Is(err, pubsub.ErrSubscriptionCancelled) {
				log.Error("Pubsub cancel, tx")
				return
			}
			if err != nil {
				log.Errorf("Pubsub: tx subscription error: %s", err)
				continue
			}
		}
	}()

	blockSub, err := blockTopic.Subscribe()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			_, err := blockSub.Next(context.Background())
			if errors.Is(err, pubsub.ErrSubscriptionCancelled) {
				log.Error("Pubsub cancel, blk")
				return
			}
			if err != nil {
				log.Errorf("Pubsub: block subscription error: %s", err)
				continue
			}
		}
	}()

	net := &Network{
		host:        host,
		connManager: cmgr,
		connGater:   conngater,
		dht:         kdht,
		pubsub:      ps,
		txTopic:     txTopic,
		blockTopic:  blockTopic,
		pstoreds:    pstoreds,
		txSub:       txSub,
		blkSub:      blockSub,
	}

	connected := func(_ inet.Network, conn inet.Conn) {
		log.Debugf("Connected to peer %s", conn.RemotePeer().String())
	}
	disconnected := func(_ inet.Network, conn inet.Conn) {
		log.Debugf("Disconnected from peer %s", conn.RemotePeer().String())
	}

	notifier := &inet.NotifyBundle{
		ConnectedF:    connected,
		DisconnectedF: disconnected,
	}

	host.Network().Notify(notifier)

	subReachability, err := host.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		return nil, err
	}

	go func(sub event.Subscription, r *dht.IpfsDHT) {
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Out():
				if !ok {
					return
				}
				if ev.(event.EvtLocalReachabilityChanged).Reachability == network.ReachabilityPublic {
					h, err := mh.Sum([]byte(RelayKey), mh.SHA2_256, -1)
					if err != nil {
						return
					}

					id := cid.NewCidV1(cid.Raw, h)
					r.Provide(ctx, id, true)
				}
			}
		}
	}(subReachability, kdht)
	return net, nil
}

func (n *Network) Close() error {
	n.txSub.Cancel()
	n.blkSub.Cancel()
	n.pstoreds.Close()
	if err := n.host.Close(); err != nil {
		return err
	}
	if err := n.connManager.Close(); err != nil {
		return err
	}
	if err := n.dht.Close(); err != nil {
		return err
	}
	return nil
}

func (n *Network) Host() host.Host {
	return n.host
}

func (n *Network) ConnManager() coreconmgr.ConnManager {
	return n.connManager
}

func (n *Network) Dht() *dht.IpfsDHT {
	return n.dht
}

func (n *Network) Pubsub() *pubsub.PubSub {
	return n.pubsub
}

func (n *Network) ConnGater() *ConnectionGater {
	return n.connGater
}

func (n *Network) SubscribeBlocks() (*pubsub.Subscription, error) {
	return n.blockTopic.Subscribe()
}

func (n *Network) SubscribeTransactions() (*pubsub.Subscription, error) {
	return n.txTopic.Subscribe()
}

func (n *Network) BroadcastBlock(blk *blocks.XThinnerBlock) error {
	ser, err := blk.Serialize()
	if err != nil {
		return err
	}
	return n.blockTopic.Publish(context.Background(), ser)
}

func (n *Network) BroadcastTransaction(tx *transactions.Transaction) error {
	ser, err := tx.Serialize()
	if err != nil {
		return err
	}
	return n.txTopic.Publish(context.Background(), ser)
}

func (n *Network) IncreaseBanscore(p peer.ID, persistent, transient uint32) {
	banned, err := n.connGater.IncreaseBanscore(p, persistent, transient)
	if err != nil {
		log.Errorf("Error setting banscore for peer %: %s", p, err)
	}
	if banned {
		n.host.Network().ClosePeer(p) //nolint:errcheck
	}
}
