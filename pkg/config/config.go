package config

type Config struct {
	// Node configuration
	NodeID        string
	ListenAddress string
	Port          int

	// Storage configuration
	StoragePath string
	MaxSize     int64

	// P2P configuration
	BootstrapPeers []string

	// API configuration
	APIPort int
}

func DefaultConfig() *Config {
	return &Config{
		ListenAddress: "0.0.0.0",
		Port:          4001,
		StoragePath:   "./storage",
		MaxSize:       1024 * 1024 * 1024 * 100, // 100GB
		APIPort:       8080,
	}
}
