package blocks

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/3FT-io/3DS/pkg/importers"
)

// Service handles block-related operations
type Service struct {
	store *Store
}

// NewService creates a new block service instance
func NewService(store *Store) *Service {
	return &Service{
		store: store,
	}
}

// ProcessModelData processes model data and stores it as blocks
func (s *Service) ProcessModelData(ctx context.Context, format string, reader io.Reader) ([]string, error) {
	// Import vertices based on format
	importer := importers.NewVertexImporter()
	var err error

	switch format {
	case "obj":
		err = importer.ImportFromOBJ(reader)
	case "fbx":
		err = importer.ImportFromFBX(reader)
	default:
		return nil, errors.New("unsupported format")
	}

	if err != nil {
		return nil, err
	}

	// Get vertices and store them as blocks
	vertices := importer.GetVertices()
	blocks := make([]string, 0, len(vertices))

	// Store each vertex as a separate block
	for _, vertex := range vertices {
		data := encodeVertex(vertex)
		hash, err := s.store.StoreBlock(ctx, data)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, hash)
	}

	return blocks, nil
}

// GetModelBlocks retrieves all blocks for a model
func (s *Service) GetModelBlocks(ctx context.Context, hashes []string) ([]*Block, error) {
	blocks := make([]*Block, 0, len(hashes))

	for _, hash := range hashes {
		block, err := s.store.GetBlock(ctx, hash)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// Helper function to encode vertex data
func encodeVertex(vertex importers.Vertex) []byte {
	// Simple encoding: just concatenate all float64 values
	// In a real implementation, you'd want to use a proper serialization format
	data := make([]byte, 8*8) // 8 float64s (3 position + 3 normal + 2 texcoord)
	binary.LittleEndian.PutUint64(data[0:8], math.Float64bits(vertex.Position[0]))
	binary.LittleEndian.PutUint64(data[8:16], math.Float64bits(vertex.Position[1]))
	binary.LittleEndian.PutUint64(data[16:24], math.Float64bits(vertex.Position[2]))
	binary.LittleEndian.PutUint64(data[24:32], math.Float64bits(vertex.Normal[0]))
	binary.LittleEndian.PutUint64(data[32:40], math.Float64bits(vertex.Normal[1]))
	binary.LittleEndian.PutUint64(data[40:48], math.Float64bits(vertex.Normal[2]))
	binary.LittleEndian.PutUint64(data[48:56], math.Float64bits(vertex.TexCoords[0]))
	binary.LittleEndian.PutUint64(data[56:64], math.Float64bits(vertex.TexCoords[1]))
	return data
}

// DeleteBlock removes a block from storage
func (s *Service) DeleteBlock(ctx context.Context, hash string) error {
	return s.store.DeleteBlock(ctx, hash)
}
