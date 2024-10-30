package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"

	"github.com/3FT-io/3DS/pkg/config"
)

const (
	ProtocolID         = "/3ds/1.0.0"
	DiscoveryNamespace = "3ds-network"
	PubsubTopic        = "3ds-messages"
	ConnectionTimeout  = 10 * time.Second
)

type Network struct {
	cfg          *config.Config
	host         host.Host
	dht          *dht.IpfsDHT
	pubsub       *pubsub.PubSub
	topic        *pubsub.Topic
	subscription *pubsub.Subscription
	peers        map[peer.ID]peer.AddrInfo
	mu           sync.RWMutex
}

func NewNetwork(cfg *config.Config) (*Network, error) {
	return &Network{
		cfg:   cfg,
		peers: make(map[peer.ID]peer.AddrInfo),
	}, nil
}

func (n *Network) Start(ctx context.Context) error {
	// Create libp2p host
	h, err := n.createHost()
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}
	n.host = h

	// Initialize DHT
	if err := n.initDHT(ctx); err != nil {
		return fmt.Errorf("failed to initialize DHT: %w", err)
	}

	// Initialize PubSub
	if err := n.initPubSub(ctx); err != nil {
		return fmt.Errorf("failed to initialize PubSub: %w", err)
	}

	// Start mDNS discovery
	if err := n.initMDNS(); err != nil {
		return fmt.Errorf("failed to initialize mDNS: %w", err)
	}

	// Connect to bootstrap peers
	if err := n.connectToBootstrapPeers(ctx); err != nil {
		return fmt.Errorf("failed to connect to bootstrap peers: %w", err)
	}

	// Start message handler
	go n.handleMessages(ctx)

	return nil
}

func (n *Network) createHost() (host.Host, error) {
	// Create multiaddr
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", n.cfg.ListenAddress, n.cfg.Port))
	if err != nil {
		return nil, err
	}

	// Create libp2p options
	opts := []libp2p.Option{
		libp2p.ListenAddrs(addr),
		libp2p.EnableNATService(),
	}

	// Only enable auto relay if we have bootstrap peers configured
	if len(n.cfg.BootstrapPeers) > 0 {
		opts = append(opts, libp2p.EnableAutoRelay())
	}

	// Create libp2p host
	return libp2p.New(opts...)
}

func (n *Network) initDHT(ctx context.Context) error {
	var err error
	n.dht, err = dht.New(ctx, n.host,
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(protocol.ID(ProtocolID)),
	)
	if err != nil {
		return err
	}

	if err := n.dht.Bootstrap(ctx); err != nil {
		return err
	}

	return nil
}

func (n *Network) initPubSub(ctx context.Context) error {
	var err error
	// Create pubsub
	n.pubsub, err = pubsub.NewGossipSub(ctx, n.host)
	if err != nil {
		return err
	}

	// Join topic
	n.topic, err = n.pubsub.Join(PubsubTopic)
	if err != nil {
		return err
	}

	// Subscribe to topic
	n.subscription, err = n.topic.Subscribe()
	if err != nil {
		return err
	}

	return nil
}

func (n *Network) initMDNS() error {
	// Create mDNS service
	service := mdns.NewMdnsService(n.host, DiscoveryNamespace, n)
	return service.Start()
}

// HandlePeerFound implements the mdns.Notifee interface
func (n *Network) HandlePeerFound(pi peer.AddrInfo) {
	n.connectToPeer(context.Background(), pi)
}

func (n *Network) connectToBootstrapPeers(ctx context.Context) error {
	for _, addr := range n.cfg.BootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			continue
		}

		if err := n.connectToPeerWithBackoff(ctx, *peerInfo); err != nil {
			continue
		}
	}
	return nil
}

func (n *Network) connectToPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	ctx, cancel := context.WithTimeout(ctx, ConnectionTimeout)
	defer cancel()

	if err := n.host.Connect(ctx, peerInfo); err != nil {
		return err
	}

	n.mu.Lock()
	n.peers[peerInfo.ID] = peerInfo
	n.mu.Unlock()

	return nil
}

func (n *Network) handleMessages(ctx context.Context) {
	for {
		msg, err := n.subscription.Next(ctx)
		if err != nil {
			// Handle context cancellation
			if ctx.Err() != nil {
				return
			}
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}

		// Handle message
		go n.processMessage(ctx, msg)
	}
}

func (n *Network) processMessage(ctx context.Context, msg *pubsub.Message) {
	// TODO: Implement message processing based on message type
	// Examples of message types:
	// - Model announcement
	// - Chunk request
	// - Chunk response
	// - Storage proof
	// - Node status
}

func (n *Network) Broadcast(ctx context.Context, data []byte) error {
	return n.topic.Publish(ctx, data)
}

func (n *Network) SendToPeer(ctx context.Context, peerID peer.ID, data []byte) error {
	stream, err := n.host.NewStream(ctx, peerID, protocol.ID(ProtocolID))
	if err != nil {
		return err
	}
	defer stream.Close()

	_, err = stream.Write(data)
	return err
}

func (n *Network) GetPeers() []peer.ID {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]peer.ID, 0, len(n.peers))
	for id := range n.peers {
		peers = append(peers, id)
	}
	return peers
}

func (n *Network) Stop() error {
	if n.subscription != nil {
		n.subscription.Cancel()
	}

	if n.topic != nil {
		n.topic.Close()
	}

	if n.dht != nil {
		if err := n.dht.Close(); err != nil {
			return err
		}
	}

	if n.host != nil {
		return n.host.Close()
	}

	return nil
}

// Message types for network communication
type MessageType int

const (
	MessageTypeModelAnnouncement MessageType = iota
	MessageTypeChunkRequest
	MessageTypeChunkResponse
	MessageTypeStorageProof
	MessageTypeNodeStatus
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
	From    peer.ID     `json:"from"`
	To      peer.ID     `json:"to,omitempty"`
}

func (n *Network) connectToPeerWithBackoff(ctx context.Context, peerInfo peer.AddrInfo) error {
	backoff := time.Second
	maxBackoff := time.Minute

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := n.connectToPeer(ctx, peerInfo)
			if err == nil {
				return nil
			}

			if backoff > maxBackoff {
				return fmt.Errorf("max backoff reached: %w", err)
			}

			time.Sleep(backoff)
			backoff *= 2
		}
	}
}

func (n *Network) GetHost() host.Host {
	return n.host
}

// ConnectToPeer exports the peer connection functionality
func (n *Network) ConnectToPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	return n.connectToPeer(ctx, peerInfo)
}

// GetSubscription returns the pubsub subscription for testing
func (n *Network) GetSubscription() *pubsub.Subscription {
	return n.subscription
}
