package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/3FT-io/3DS/pkg/api"
	"github.com/3FT-io/3DS/pkg/config"
	"github.com/3FT-io/3DS/pkg/core"
	"github.com/3FT-io/3DS/pkg/p2p"
)

func main() {
	cfg := config.DefaultConfig()

	node, err := core.NewNode(cfg)
	if err != nil {
		log.Fatal(err)
	}

	network, err := p2p.NewNetwork(cfg)
	if err != nil {
		log.Fatal(err)
	}

	storage, err := core.NewStorage(cfg.StoragePath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := node.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Initialize API
	api, err := api.NewAPI(node, network, storage, cfg.APIPort)
	if err != nil {
		log.Fatal(err)
	}

	// Start API server
	go func() {
		if err := api.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	if err := node.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// During shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := api.Stop(ctx); err != nil {
		log.Printf("Error shutting down API server: %v", err)
	}
}
