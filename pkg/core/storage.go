package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const ChunkSize = 1024 * 1024 * 5 // 5MB chunks

type Storage struct {
	basePath string
	metadata map[string]*ModelMetadata
	mu       sync.RWMutex
}

// StorageStatus represents the current state of the storage system
type StorageStatus struct {
	TotalModels int            `json:"total_models"`
	TotalSize   int64          `json:"total_size"`
	BasePath    string         `json:"base_path"`
	Models      []ModelSummary `json:"models"`
}

type ModelSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStorage(path string) (*Storage, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	return &Storage{
		basePath: path,
		metadata: make(map[string]*ModelMetadata),
	}, nil
}

func (s *Storage) StoreModel(ctx context.Context, name string, format string, reader io.Reader) (*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create model metadata
	metadata := &ModelMetadata{
		ID:   generateUUID(),
		Name: name,

		Format:    format,
		CreatedAt: time.Now(),
	}

	// Create model directory
	modelPath := filepath.Join(s.basePath, metadata.ID)
	if err := os.MkdirAll(modelPath, 0755); err != nil {
		return nil, err
	}

	// Split into chunks and store
	chunks, size, err := s.splitAndStoreChunks(ctx, modelPath, reader)
	if err != nil {
		// Clean up on error
		os.RemoveAll(modelPath)
		return nil, err
	}

	metadata.Chunks = chunks
	metadata.Size = size
	metadata.Hash = metadata.CalculateHash()

	// Store metadata in memory
	s.metadata[metadata.ID] = metadata

	return metadata, nil
}

func (s *Storage) splitAndStoreChunks(ctx context.Context, modelPath string, reader io.Reader) ([]string, int64, error) {
	var chunks []string
	var totalSize int64

	buffer := make([]byte, ChunkSize)
	chunkIndex := 0

	for {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
			n, err := reader.Read(buffer)
			if err != nil && err != io.EOF {
				return nil, 0, err
			}
			if err == io.EOF {
				return chunks, totalSize, nil
			}

			chunk := &ModelChunk{
				ID:    generateUUID(),
				Data:  buffer[:n],
				Index: chunkIndex,
			}

			// Store chunk
			chunkPath := filepath.Join(modelPath, fmt.Sprintf("chunk_%d", chunkIndex))
			if err := s.storeChunk(chunk, chunkPath); err != nil {
				return nil, 0, err
			}

			chunks = append(chunks, chunk.ID)
			totalSize += int64(n)
			chunkIndex++
		}
	}
}

func (s *Storage) storeChunk(chunk *ModelChunk, path string) error {
	return os.WriteFile(path, chunk.Data, 0644)
}

func generateUUID() string {
	return uuid.New().String()
}

func (s *Storage) ListModels(ctx context.Context) ([]ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	models := make([]ModelMetadata, 0, len(s.metadata))
	for _, model := range s.metadata {
		models = append(models, *model)
	}

	return models, nil
}

// GetModelMetadata retrieves metadata for a specific model by ID
func (s *Storage) GetModelMetadata(ctx context.Context, modelID string) (interface{}, error) {
	// Assuming you store metadata alongside your models
	// Implementation will depend on your storage backend
	metadata, err := s.getMetadata(modelID) // You'll need to implement this internal method
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

func (s *Storage) getMetadata(modelID string) (*ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.metadata[modelID]
	if !exists {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}
	return metadata, nil
}

func (s *Storage) DeleteModel(ctx context.Context, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if model exists
	if _, exists := s.metadata[modelID]; !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	// Delete from metadata map
	delete(s.metadata, modelID)

	// Delete model directory
	modelPath := filepath.Join(s.basePath, modelID)
	return os.RemoveAll(modelPath)
}

// GetStatus returns the current status of the storage system
func (s *Storage) GetStatus(ctx context.Context) (*StorageStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &StorageStatus{
		TotalModels: len(s.metadata),
		BasePath:    s.basePath,
		Models:      make([]ModelSummary, 0, len(s.metadata)),
	}

	for _, model := range s.metadata {
		status.TotalSize += model.Size
		status.Models = append(status.Models, ModelSummary{
			ID:        model.ID,
			Name:      model.Name,
			Size:      model.Size,
			CreatedAt: model.CreatedAt,
		})
	}

	return status, nil
}

// StreamModel reads a model's chunks and streams them to the provided writer
func (s *Storage) StreamModel(ctx context.Context, modelID string, writer io.Writer) error {
	metadata, err := s.getMetadata(modelID)
	if err != nil {
		return err
	}

	modelPath := filepath.Join(s.basePath, modelID)

	// Read and stream each chunk in order
	for i := 0; i < len(metadata.Chunks); i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunkPath := filepath.Join(modelPath, fmt.Sprintf("chunk_%d", i))
			data, err := os.ReadFile(chunkPath)
			if err != nil {
				return fmt.Errorf("failed to read chunk %d: %w", i, err)
			}

			if _, err := writer.Write(data); err != nil {
				return fmt.Errorf("failed to write chunk %d: %w", i, err)
			}
		}
	}

	return nil
}

// GetModel retrieves a model's metadata by ID
func (s *Storage) GetModel(ctx context.Context, modelID string) (*ModelMetadata, error) {
	metadata, err := s.getMetadata(modelID)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}
