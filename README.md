# 3DS (3D Storage System)

3DS is a decentralized 3D model storage and distribution system built with Go. It provides a P2P network for storing and sharing 3D models with built-in content addressing and efficient chunked storage.

## Features

- **Decentralized Storage**: Store 3D models across a distributed P2P network
- **Multiple Format Support**: Handles various 3D model formats (GLTF, GLB, OBJ, FBX)
- **Chunked Storage**: Large models are automatically split into manageable chunks
- **RESTful API**: Simple HTTP API for model management and network operations
- **P2P Network**: Built on libp2p with DHT-based peer discovery
- **CORS Support**: Built-in CORS support for web applications
- **Health Monitoring**: Endpoints for monitoring system and network health

## Installation

```bash
go get github.com/3FT-io/3DS
```

## Quick Start

1. Build the project:

```bash
go build -o 3ds cmd/3ds/main.go
```

2. Run the server:

```bash
./3ds
```

The server will start on port 8080 by default.

## API Endpoints

### Model Management
- `POST /models` - Upload a new 3D model
- `GET /models` - List all available models
- `GET /models/{id}` - Download a specific model
- `DELETE /models/{id}` - Delete a model
- `GET /models/{id}/metadata` - Get model metadata

### Network Operations
- `GET /network/status` - Get network status
- `GET /network/peers` - List connected peers

### System Status
- `GET /health` - Check system health
- `GET /storage/status` - Get storage system status

## Configuration

Default configuration can be modified through environment variables or by creating a custom config:

```go
config := &config.Config{
ListenAddress: "0.0.0.0",
Port: 4001,
StoragePath: "./storage",
MaxSize: 1024 1024 1024 100, // 100GB
APIPort: 8080,
}
```

## Architecture

3DS consists of several core components:

- **API Server**: RESTful interface for client interactions
- **Storage System**: Handles model storage and chunking
- **P2P Network**: Manages peer connections and data distribution
- **Node**: Coordinates between components and maintains system state

## Development

### Prerequisites

- Go 1.16 or higher
- libp2p dependencies

### Building from Source

```bash
git clone https://github.com/3FT-io/3DS.git
cd 3DS
go build ./...
```

### Running Tests

```bash
go test ./... -v
```


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contact

contact@3ft.io