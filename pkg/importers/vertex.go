package importers

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Vertex represents a 3D vertex with position, normal, and texture coordinates
type Vertex struct {
	Position  [3]float64
	Normal    [3]float64
	TexCoords [2]float64
}

// VertexImporter handles importing vertices from different 3D model formats
type VertexImporter struct {
	vertices []Vertex
}

// NewVertexImporter creates a new vertex importer instance
func NewVertexImporter() *VertexImporter {
	return &VertexImporter{
		vertices: make([]Vertex, 0),
	}
}

// ImportFromOBJ imports vertices from OBJ format
func (vi *VertexImporter) ImportFromOBJ(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)

	var positions [][3]float64
	var normals [][3]float64
	var texCoords [][2]float64

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "v": // Vertex position
			if len(fields) < 4 {
				return errors.New("invalid vertex position")
			}
			pos, err := parseVector3(fields[1:4])
			if err != nil {
				return fmt.Errorf("failed to parse vertex position: %w", err)
			}
			positions = append(positions, pos)

		case "vn": // Vertex normal
			if len(fields) < 4 {
				return errors.New("invalid vertex normal")
			}
			normal, err := parseVector3(fields[1:4])
			if err != nil {
				return fmt.Errorf("failed to parse vertex normal: %w", err)
			}
			normals = append(normals, normal)

		case "vt": // Texture coordinates
			if len(fields) < 3 {
				return errors.New("invalid texture coordinates")
			}
			tex, err := parseVector2(fields[1:3])
			if err != nil {
				return fmt.Errorf("failed to parse texture coordinates: %w", err)
			}
			texCoords = append(texCoords, tex)

		case "f": // Face
			if err := vi.processFace(fields[1:], positions, normals, texCoords); err != nil {
				return fmt.Errorf("failed to process face: %w", err)
			}
		}
	}

	return scanner.Err()
}

// ImportFromFBX imports vertices from FBX format
func (vi *VertexImporter) ImportFromFBX(reader io.Reader) error {
	// Read the entire FBX file
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read FBX file: %w", err)
	}

	// Parse FBX binary format
	vertices, normals, uvs, err := parseFBXBinary(data)
	if err != nil {
		return fmt.Errorf("failed to parse FBX binary: %w", err)
	}

	// Process vertices
	for i := 0; i < len(vertices); i += 3 {
		vertex := Vertex{
			Position: [3]float64{
				vertices[i],
				vertices[i+1],
				vertices[i+2],
			},
		}

		// Add normal if available
		if i < len(normals) {
			vertex.Normal = [3]float64{
				normals[i],
				normals[i+1],
				normals[i+2],
			}
		}

		// Add UV coordinates if available
		if i/3*2 < len(uvs) {
			vertex.TexCoords = [2]float64{
				uvs[i/3*2],
				uvs[i/3*2+1],
			}
		}

		vi.vertices = append(vi.vertices, vertex)
	}

	return nil
}

// Helper function to parse FBX binary format
func parseFBXBinary(data []byte) (vertices, normals, uvs []float64, err error) {
	// FBX Binary format magic number "Kaydara FBX Binary  "
	magic := []byte("Kaydara FBX Binary  ")
	if len(data) < len(magic) || string(data[:len(magic)]) != string(magic) {
		return nil, nil, nil, errors.New("invalid FBX binary format")
	}

	// Parse FBX version (located at offset 23)
	version := binary.LittleEndian.Uint32(data[23:27])
	if version < 7100 {
		return nil, nil, nil, fmt.Errorf("unsupported FBX version: %d", version)
	}

	// Parse geometry data
	// This is a simplified implementation - real FBX parsing would need to handle
	// the full node structure and property types
	vertices, normals, uvs = extractGeometryData(data[27:])

	return vertices, normals, uvs, nil
}

// GetVertices returns the imported vertices
func (vi *VertexImporter) GetVertices() []Vertex {
	return vi.vertices
}

// Helper functions for vector parsing
func parseVector3(values []string) ([3]float64, error) {
	if len(values) < 3 {
		return [3]float64{}, errors.New("not enough values for vector3")
	}

	var result [3]float64
	for i := 0; i < 3; i++ {
		val, err := strconv.ParseFloat(values[i], 64)
		if err != nil {
			return [3]float64{}, err
		}
		result[i] = val
	}
	return result, nil
}

func parseVector2(values []string) ([2]float64, error) {
	if len(values) < 2 {
		return [2]float64{}, errors.New("not enough values for vector2")
	}

	var result [2]float64
	for i := 0; i < 2; i++ {
		val, err := strconv.ParseFloat(values[i], 64)
		if err != nil {
			return [2]float64{}, err
		}
		result[i] = val
	}
	return result, nil
}

// processFace handles OBJ face definitions and creates vertices
func (vi *VertexImporter) processFace(faceData []string, positions [][3]float64, normals [][3]float64, texCoords [][2]float64) error {
	if len(faceData) < 3 {
		return errors.New("face must have at least 3 vertices")
	}

	// Process each vertex in the face
	for _, vertexData := range faceData {
		// Split vertex data into position/texcoord/normal indices
		indices := strings.Split(vertexData, "/")

		// Parse position index (required)
		posIndex, err := parseIndex(indices[0], len(positions))
		if err != nil {
			return fmt.Errorf("invalid position index: %w", err)
		}

		vertex := Vertex{
			Position: positions[posIndex],
		}

		// Parse texture coordinate index (optional)
		if len(indices) > 1 && indices[1] != "" {
			texIndex, err := parseIndex(indices[1], len(texCoords))
			if err != nil {
				return fmt.Errorf("invalid texture coordinate index: %w", err)
			}
			vertex.TexCoords = texCoords[texIndex]
		}

		// Parse normal index (optional)
		if len(indices) > 2 && indices[2] != "" {
			normalIndex, err := parseIndex(indices[2], len(normals))
			if err != nil {
				return fmt.Errorf("invalid normal index: %w", err)
			}
			vertex.Normal = normals[normalIndex]
		}

		vi.vertices = append(vi.vertices, vertex)
	}

	return nil
}

// parseIndex converts a 1-based OBJ index to a 0-based array index
func parseIndex(indexStr string, maxLen int) (int, error) {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return 0, err
	}

	// Handle negative indices (relative to end of array)
	if index < 0 {
		index = maxLen + index
	} else {
		// Convert from 1-based to 0-based indexing
		index--
	}

	// Validate index bounds
	if index < 0 || index >= maxLen {
		return 0, fmt.Errorf("index out of bounds: %d (max: %d)", index, maxLen)
	}

	return index, nil
}

// extractGeometryData parses FBX binary data to extract geometry information
func extractGeometryData(data []byte) (vertices, normals, uvs []float64) {
	// Initialize slices to store the geometry data
	vertices = make([]float64, 0)
	normals = make([]float64, 0)
	uvs = make([]float64, 0)

	// FBX uses a node structure. We need to find the Geometry node
	// and its child nodes for vertices, normals, and UV coordinates
	offset := uint32(0)
	for offset < uint32(len(data)) {
		// Check if we have enough data left to read a node header
		if offset+4 > uint32(len(data)) {
			break
		}

		// Read node header (endOffset, numProperties, propertyListLen)
		endOffset := binary.LittleEndian.Uint32(data[offset : offset+4])
		if endOffset == 0 || endOffset > uint32(len(data)) {
			break
		}

		// Skip header
		offset += 13 // Standard FBX node header size

		// Read node name length
		nameLen := uint8(data[offset])
		offset++

		// Read node name
		if offset+uint32(nameLen) > uint32(len(data)) {
			break
		}
		nodeName := string(data[offset : offset+uint32(nameLen)])
		offset += uint32(nameLen)

		// Process node based on its name
		switch nodeName {
		case "Vertices":
			vertices = extractFloatArray(data[offset:endOffset])
		case "Normals":
			normals = extractFloatArray(data[offset:endOffset])
		case "UV":
			uvs = extractFloatArray(data[offset:endOffset])
		}

		// Move to next node
		offset = endOffset
	}

	return
}

// extractFloatArray reads an array of float64 values from FBX binary data
func extractFloatArray(data []byte) []float64 {
	result := make([]float64, 0)

	// First 4 bytes contain the array length
	if len(data) < 4 {
		return result
	}

	arrayLen := binary.LittleEndian.Uint32(data[0:4])
	offset := uint32(4)

	// Read each float64 value
	for i := uint32(0); i < arrayLen && offset+8 <= uint32(len(data)); i++ {
		bits := binary.LittleEndian.Uint64(data[offset : offset+8])
		value := math.Float64frombits(bits)
		result = append(result, value)
		offset += 8
	}

	return result
}
