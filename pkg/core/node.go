package core

import (
	"context"

	"github.com/3FT-io/3DS/pkg/config"
	"github.com/3FT-io/3DS/pkg/p2p"
)

type Node struct {
	config  *config.Config
	storage *Storage
	network *p2p.Network
}

func NewNode(cfg *config.Config) (*Node, error) {
	storage, err := NewStorage(cfg.StoragePath)
	if err != nil {
		return nil, err
	}

	network, err := p2p.NewNetwork(cfg)
	if err != nil {
		return nil, err
	}

	return &Node{
		config:  cfg,
		storage: storage,
		network: network,
	}, nil
}

func (n *Node) Start(ctx context.Context) error {
	// Start P2P network
	if err := n.network.Start(ctx); err != nil {
		return err
	}

	// Start discovery service
	go n.discovery(ctx)

	// Start storage maintenance
	go n.maintenance(ctx)

	return nil
}

func (n *Node) Stop() error {
	return n.network.Stop()
}

func (n *Node) discovery(ctx context.Context) {
	// Implement peer discovery logic
}

func (n *Node) maintenance(ctx context.Context) {
	// Implement storage maintenance logic
}
