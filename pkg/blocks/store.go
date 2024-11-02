package blocks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Block represents a chunk of 3D model data
type Block struct {
	Hash string
	Size int64
	Data []byte
}

// Store manages the storage of model data blocks
type Store struct {
	basePath string
	mu       sync.RWMutex
}

// NewStore creates a new block store instance
func NewStore(basePath string) (*Store, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}

	return &Store{
		basePath: basePath,
	}, nil
}

// StoreBlock stores a block of data and returns its hash
func (s *Store) StoreBlock(ctx context.Context, data []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate hash
	hash := calculateHash(data)
	blockPath := s.getBlockPath(hash)

	// Check if block already exists
	if _, err := os.Stat(blockPath); err == nil {
		return hash, nil
	}

	// Create block file
	if err := os.WriteFile(blockPath, data, 0644); err != nil {
		return "", err
	}

	return hash, nil
}

// GetBlock retrieves a block by its hash
func (s *Store) GetBlock(ctx context.Context, hash string) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	blockPath := s.getBlockPath(hash)
	data, err := os.ReadFile(blockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("block not found")
		}
		return nil, err
	}

	return &Block{
		Hash: hash,
		Size: int64(len(data)),
		Data: data,
	}, nil
}

// DeleteBlock removes a block from storage
func (s *Store) DeleteBlock(ctx context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	blockPath := s.getBlockPath(hash)
	return os.Remove(blockPath)
}

func (s *Store) getBlockPath(hash string) string {
	return filepath.Join(s.basePath, hash)
}

func calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
