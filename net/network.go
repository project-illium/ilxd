// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	coreconmgr "github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
)

const (
	BlockTopic        = "blocks"
	TransactionsTopic = "transactions"
)

type Network struct {
	host        host.Host
	connManager coreconmgr.ConnManager
	connGater   *ConnectionGater
	dht         *dht.IpfsDHT
	pubsub      *pubsub.PubSub
	txTopic     *pubsub.Topic
	blockTopic  *pubsub.Topic
}

func NewNetwork(ctx context.Context, opts ...Option) (*Network, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	var (
		idht *dht.IpfsDHT
		cmgr coreconmgr.ConnManager = connmgr.NewConnManager(
			100,         // Lowwater
			400,         // HighWater,
			time.Minute, // GracePeriod
		)
		pstore peerstore.Peerstore
		err    error
	)

	if cfg.host == nil {
		pstore, err = pstoreds.NewPeerstore(ctx, cfg.datastore, pstoreds.DefaultOpts())
		if err != nil {
			return nil, err
		}
	}

	conngater, err := NewConnectionGater(cfg.datastore, pstore, cfg.banDuration, cfg.maxBanscore)
	if err != nil {
		return nil, err
	}

	dhtOpts := []dht.Option{
		dht.DisableValues(),
		dht.ProtocolPrefix(cfg.params.ProtocolPrefix),
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
			idht, err = dht.New(ctx, h, dhtOpts...)
			return idht, err
		}),

		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		libp2p.EnableAutoRelay(),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),

		libp2p.UserAgent(cfg.userAgent),

		libp2p.Ping(true),

		libp2p.Peerstore(pstore),

		libp2p.DisableRelay(),

		libp2p.EnableHolePunching(),

		libp2p.ConnectionGater(conngater),
	)

	if !cfg.disableNatPortMap {
		hostOpts = libp2p.ChainOptions(libp2p.NATPortMap(), hostOpts)
	}

	var host host.Host
	if cfg.host != nil {
		host = cfg.host
		cmgr = host.ConnManager()
		pstore = host.Peerstore()
		idht, err = dht.New(ctx, cfg.host, dhtOpts...)
		if err != nil {
			return nil, err
		}
	} else {
		host, err = libp2p.New(hostOpts)
		if err != nil {
			return nil, err
		}
	}

	for i, pid := range pstore.Peers() {
		pi := pstore.PeerInfo(pid)
		host.Connect(ctx, pi)
		if i > 50 {
			break
		}
	}

	// The last step to get fully up and running would be to connect to
	// seed peers (or any other peers). We leave this commented as
	// this is an example and the peer will die as soon as it finishes, so
	// it is unnecessary to put strain on the network.
	for _, addr := range cfg.seedAddrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("%w: malformatted seed peer", ErrNetworkConfig)
		}

		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return nil, err
		}
		// We ignore errors as some bootstrap peers may be down
		// and that is fine.
		host.Connect(ctx, *pi)
	}

	// create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(
		ctx,
		host,
		pubsub.WithNoAuthor(),
		pubsub.WithDiscovery(discovery.NewRoutingDiscovery(idht)),
		pubsub.WithMaxMessageSize(1<<23),
		pubsub.WithMessageIdFn(func(pmsg *pb.Message) string {
			h := hash.HashFunc(pmsg.Data)
			return string(h[:])
		}),
		pubsub.WithGossipSubProtocols([]protocol.ID{cfg.params.ProtocolPrefix + pubsub.GossipSubID_v11}, func(feature pubsub.GossipSubFeature, id protocol.ID) bool {
			if id == cfg.params.ProtocolPrefix+pubsub.GossipSubID_v11 && (feature == pubsub.GossipSubFeatureMesh || feature == pubsub.GossipSubFeaturePX) {
				return true
			}
			return false
		}),
	)
	if err != nil {
		return nil, err
	}

	err = ps.RegisterTopicValidator(TransactionsTopic, pubsub.ValidatorEx(func(ctx context.Context, p peer.ID, m *pubsub.Message) pubsub.ValidationResult {
		var tx transactions.Transaction
		if err := tx.Deserialize(m.Data); err != nil {
			return pubsub.ValidationReject
		}
		err := cfg.acceptToMempool(&tx)
		switch err.(type) {
		case mempool.PolicyError:
			// Policy errors do no penalize peer
			return pubsub.ValidationIgnore
		case blockchain.RuleError:
			// Rule errors do
			return pubsub.ValidationReject
		case nil:
			return pubsub.ValidationAccept
		default:
			return pubsub.ValidationIgnore
		}
	}))
	if err != nil {
		return nil, err
	}

	// For blocks we will wait for the full block to be recovered from the compact block
	// so that we can validate it before returning here.
	err = ps.RegisterTopicValidator(BlockTopic, pubsub.ValidatorEx(func(ctx context.Context, p peer.ID, m *pubsub.Message) pubsub.ValidationResult {
		var blk blocks.XThinnerBlock
		if err := blk.Deserialize(m.Data); err != nil {
			return pubsub.ValidationReject
		}
		err := cfg.validateBlock(&blk, p)
		switch err.(type) {
		case blockchain.OrphanBlockError:
			// Orphans we won't relay (yet) but won't penalize them either.
			return pubsub.ValidationIgnore
		case blockchain.RuleError:
			// Rule errors do
			return pubsub.ValidationReject
		case nil:
			return pubsub.ValidationAccept
		default:
			return pubsub.ValidationIgnore
		}
	}))
	if err != nil {
		return nil, err
	}

	txTopic, err := ps.Join(TransactionsTopic)
	if err != nil {
		return nil, err
	}

	blockTopic, err := ps.Join(BlockTopic)
	if err != nil {
		return nil, err
	}

	net := &Network{
		host:        host,
		connManager: cmgr,
		connGater:   conngater,
		dht:         idht,
		pubsub:      ps,
		txTopic:     txTopic,
		blockTopic:  blockTopic,
	}

	connected := func(_ inet.Network, conn inet.Conn) {
		log.Debugf("Connect to peer %s", conn.RemotePeer().String())
	}
	disconnected := func(_ inet.Network, conn inet.Conn) {
		log.Debugf("Disconnect from peer %s", conn.RemotePeer().String())
	}

	notifier := &inet.NotifyBundle{
		ConnectedF:    connected,
		DisconnectedF: disconnected,
	}

	host.Network().Notify(notifier)

	return net, nil
}

func (n *Network) Close() error {
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

func (n *Network) Routing() routing.Routing {
	return n.dht
}

func (n *Network) Pubsub() *pubsub.PubSub {
	return n.pubsub
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
		n.host.Network().ClosePeer(p)
	}
}
