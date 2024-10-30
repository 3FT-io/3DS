package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/3FT-io/3DS/pkg/config"
	"github.com/3FT-io/3DS/pkg/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestNetwork(t *testing.T) (*p2p.Network, func()) {
	cfg := &config.Config{
		ListenAddress: "127.0.0.1",
		Port:          0, // Use random port
	}

	network, err := p2p.NewNetwork(cfg)
	require.NoError(t, err)

	cleanup := func() {
		network.Stop()
	}

	return network, cleanup
}

func TestNetworkStartStop(t *testing.T) {
	network, cleanup := setupTestNetwork(t)
	defer cleanup()

	ctx := context.Background()
	err := network.Start(ctx)
	require.NoError(t, err)

	// Verify network is running
	assert.NotNil(t, network.GetHost())
	assert.NotEmpty(t, network.GetHost().Addrs())

	// Stop network
	err = network.Stop()
	require.NoError(t, err)
}

func TestPeerConnection(t *testing.T) {
	// Create two networks
	network1, cleanup1 := setupTestNetwork(t)
	defer cleanup1()

	network2, cleanup2 := setupTestNetwork(t)
	defer cleanup2()

	ctx := context.Background()

	// Start both networks
	require.NoError(t, network1.Start(ctx))
	require.NoError(t, network2.Start(ctx))

	// Connect network2 to network1
	peerInfo := network1.GetHost().Peerstore().PeerInfo(network1.GetHost().ID())

	err := network2.ConnectToPeer(ctx, peerInfo)
	require.NoError(t, err)

	// Verify connection
	time.Sleep(time.Second) // Allow time for connection
	peers := network2.GetPeers()
	assert.Contains(t, peers, network1.GetHost().ID())
}

func TestMessageBroadcast(t *testing.T) {
	network1, cleanup1 := setupTestNetwork(t)
	defer cleanup1()

	network2, cleanup2 := setupTestNetwork(t)
	defer cleanup2()

	ctx := context.Background()

	// Start both networks and connect them
	require.NoError(t, network1.Start(ctx))
	require.NoError(t, network2.Start(ctx))

	peerInfo := network1.GetHost().Peerstore().PeerInfo(network1.GetHost().ID())
	require.NoError(t, network2.ConnectToPeer(ctx, peerInfo))

	// Broadcast message
	testMessage := []byte("test message")
	err := network1.Broadcast(ctx, testMessage)
	require.NoError(t, err)

	// TODO: Add message reception verification
	// This would require implementing a message handler and verification mechanism
}
