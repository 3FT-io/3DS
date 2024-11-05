package core

import (
	"bytes"
	"context"
	"time"

	"github.com/3FT-io/3DS/pkg/blocks"
)

// Object represents a 3D model object with its metadata and blocks
type Object struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Format      string    `json:"format"`
	Size        int64     `json:"size"`
	BlockHashes []string  `json:"blocks"`
	Materials   []string  `json:"materials,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ObjectService handles operations on 3D model objects
type ObjectService struct {
	blockService *blocks.Service
	storage      *Storage
}

// NewObjectService creates a new object service instance
func NewObjectService(blockService *blocks.Service, storage *Storage) *ObjectService {
	return &ObjectService{
		blockService: blockService,
		storage:      storage,
	}
}

// CreateObject creates a new 3D model object
func (s *ObjectService) CreateObject(ctx context.Context, name, format string, data []byte) (*Object, error) {
	// Process model data into blocks
	blockHashes, err := s.blockService.ProcessModelData(ctx, format, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Create object metadata
	obj := &Object{
		Name:        name,
		Format:      format,
		Size:        int64(len(data)),
		BlockHashes: blockHashes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store object metadata
	if err := s.storage.StoreObject(ctx, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// GetObject retrieves an object by ID
func (s *ObjectService) GetObject(ctx context.Context, id string) (*Object, error) {
	return s.storage.GetObject(ctx, id)
}

// DeleteObject removes an object and its blocks
func (s *ObjectService) DeleteObject(ctx context.Context, id string) error {
	obj, err := s.storage.GetObject(ctx, id)
	if err != nil {
		return err
	}

	// Delete all blocks
	for _, hash := range obj.BlockHashes {
		if err := s.blockService.DeleteBlock(ctx, hash); err != nil {
			return err
		}
	}

	// Delete object metadata
	return s.storage.DeleteObject(ctx, id)
}
