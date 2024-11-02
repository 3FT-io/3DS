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

// Material represents material properties for 3D models
type Material struct {
	Name          string
	AmbientColor  [3]float64
	DiffuseColor  [3]float64
	SpecularColor [3]float64
	Shininess     float64
	DiffuseMap    string
	NormalMap     string
	SpecularMap   string
	Transparency  float64
}

// MaterialImporter handles importing materials from different 3D model formats
type MaterialImporter struct {
	materials map[string]*Material
}

// NewMaterialImporter creates a new material importer instance
func NewMaterialImporter() *MaterialImporter {
	return &MaterialImporter{
		materials: make(map[string]*Material),
	}
}

// ImportFromOBJ imports materials from MTL format (OBJ materials)
func (mi *MaterialImporter) ImportFromOBJ(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	var currentMaterial *Material

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
		case "newmtl":
			name := strings.Join(fields[1:], " ")
			currentMaterial = &Material{Name: name}
			mi.materials[name] = currentMaterial

		case "Ka":
			if currentMaterial == nil {
				return errors.New("ambient color specified before material")
			}
			color, err := ParseVector3(fields[1:])
			if err != nil {
				return fmt.Errorf("invalid ambient color: %w", err)
			}
			currentMaterial.AmbientColor = color

		case "Kd":
			if currentMaterial == nil {
				return errors.New("diffuse color specified before material")
			}
			color, err := ParseVector3(fields[1:])
			if err != nil {
				return fmt.Errorf("invalid diffuse color: %w", err)
			}
			currentMaterial.DiffuseColor = color

		case "Ks":
			if currentMaterial == nil {
				return errors.New("specular color specified before material")
			}
			color, err := ParseVector3(fields[1:])
			if err != nil {
				return fmt.Errorf("invalid specular color: %w", err)
			}
			currentMaterial.SpecularColor = color

		case "Ns":
			if currentMaterial == nil {
				return errors.New("shininess specified before material")
			}
			value, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return fmt.Errorf("invalid shininess value: %w", err)
			}
			currentMaterial.Shininess = value

		case "d", "Tr":
			if currentMaterial == nil {
				return errors.New("transparency specified before material")
			}
			value, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return fmt.Errorf("invalid transparency value: %w", err)
			}
			if fields[0] == "Tr" {
				value = 1.0 - value // Convert from Tr (transparency) to d (dissolve)
			}
			currentMaterial.Transparency = value

		case "map_Kd":
			if currentMaterial == nil {
				return errors.New("diffuse map specified before material")
			}
			currentMaterial.DiffuseMap = strings.Join(fields[1:], " ")

		case "map_Bump", "bump":
			if currentMaterial == nil {
				return errors.New("normal map specified before material")
			}
			currentMaterial.NormalMap = strings.Join(fields[1:], " ")

		case "map_Ks":
			if currentMaterial == nil {
				return errors.New("specular map specified before material")
			}
			currentMaterial.SpecularMap = strings.Join(fields[1:], " ")
		}
	}

	return scanner.Err()
}

// ImportFromFBX imports materials from FBX format
func (mi *MaterialImporter) ImportFromFBX(reader io.Reader) error {
	// Read the entire FBX file
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read FBX file: %w", err)
	}

	// Parse FBX binary format
	materials, err := parseFBXMaterials(data)
	if err != nil {
		return fmt.Errorf("failed to parse FBX materials: %w", err)
	}

	// Store the materials
	for name, mat := range materials {
		mi.materials[name] = mat
	}

	return nil
}

// GetMaterial returns a material by name
func (mi *MaterialImporter) GetMaterial(name string) (*Material, bool) {
	mat, ok := mi.materials[name]
	return mat, ok
}

// GetMaterials returns all imported materials
func (mi *MaterialImporter) GetMaterials() map[string]*Material {
	return mi.materials
}

// parseFBXMaterials extracts materials from FBX binary data
func parseFBXMaterials(data []byte) (map[string]*Material, error) {
	// Check FBX magic number and version
	magic := []byte("Kaydara FBX Binary  ")
	if len(data) < len(magic) || string(data[:len(magic)]) != string(magic) {
		return nil, errors.New("invalid FBX binary format")
	}

	// Parse FBX version (located at offset 23)
	version := binary.LittleEndian.Uint32(data[23:27])
	if version < 7100 {
		return nil, fmt.Errorf("unsupported FBX version: %d", version)
	}

	materials := make(map[string]*Material)
	offset := uint32(27) // Start after header and version

	for offset < uint32(len(data)) {
		// Check if we have enough data left to read a node header
		if offset+4 > uint32(len(data)) {
			break
		}

		// Read node header
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

		// Process material nodes
		if nodeName == "Material" {
			material, err := parseMaterialNode(data[offset:endOffset])
			if err != nil {
				return nil, err
			}
			materials[material.Name] = material
		}

		// Move to next node
		offset = endOffset
	}

	return materials, nil
}

// parseMaterialNode parses a single material node from FBX data
func parseMaterialNode(data []byte) (*Material, error) {
	material := &Material{}
	offset := uint32(0)

	for offset < uint32(len(data)) {
		// Read property header
		if offset+4 > uint32(len(data)) {
			break
		}

		propLen := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Read property name
		nameLen := uint8(data[offset])
		offset++

		if offset+uint32(nameLen) > uint32(len(data)) {
			break
		}
		propName := string(data[offset : offset+uint32(nameLen)])
		offset += uint32(nameLen)

		// Parse property value based on name
		switch propName {
		case "Name":
			material.Name = string(data[offset : offset+propLen])
		case "AmbientColor":
			material.AmbientColor = parseColor(data[offset : offset+24])
		case "DiffuseColor":
			material.DiffuseColor = parseColor(data[offset : offset+24])
		case "SpecularColor":
			material.SpecularColor = parseColor(data[offset : offset+24])
		case "Shininess":
			material.Shininess = math.Float64frombits(binary.LittleEndian.Uint64(data[offset : offset+8]))
		}

		offset += propLen
	}

	return material, nil
}

// parseColor converts FBX color data to [3]float64
func parseColor(data []byte) [3]float64 {
	return [3]float64{
		math.Float64frombits(binary.LittleEndian.Uint64(data[0:8])),
		math.Float64frombits(binary.LittleEndian.Uint64(data[8:16])),
		math.Float64frombits(binary.LittleEndian.Uint64(data[16:24])),
	}
}
