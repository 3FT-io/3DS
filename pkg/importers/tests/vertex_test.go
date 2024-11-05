package importers_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/3FT-io/3DS/pkg/importers"
)

func TestImportFromOBJ(t *testing.T) {
	tests := []struct {
		name     string
		objData  string
		expected []importers.Vertex
		wantErr  bool
	}{
		{
			name: "simple triangle",
			objData: `
v 0.0 0.0 0.0
v 1.0 0.0 0.0
v 0.0 1.0 0.0
f 1 2 3
`,
			expected: []importers.Vertex{
				{Position: [3]float64{0.0, 0.0, 0.0}},
				{Position: [3]float64{1.0, 0.0, 0.0}},
				{Position: [3]float64{0.0, 1.0, 0.0}},
			},
			wantErr: false,
		},
		{
			name: "triangle with normals and texcoords",
			objData: `
v 0.0 0.0 0.0
v 1.0 0.0 0.0
v 0.0 1.0 0.0
vn 0.0 0.0 1.0
vn 0.0 0.0 1.0
vn 0.0 0.0 1.0
vt 0.0 0.0
vt 1.0 0.0
vt 0.0 1.0
f 1/1/1 2/2/2 3/3/3
`,
			expected: []importers.Vertex{
				{
					Position:  [3]float64{0.0, 0.0, 0.0},
					Normal:    [3]float64{0.0, 0.0, 1.0},
					TexCoords: [2]float64{0.0, 0.0},
				},
				{
					Position:  [3]float64{1.0, 0.0, 0.0},
					Normal:    [3]float64{0.0, 0.0, 1.0},
					TexCoords: [2]float64{1.0, 0.0},
				},
				{
					Position:  [3]float64{0.0, 1.0, 0.0},
					Normal:    [3]float64{0.0, 0.0, 1.0},
					TexCoords: [2]float64{0.0, 1.0},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid vertex",
			objData: `
v 0.0 0.0
v 1.0 0.0 0.0
v 0.0 1.0 0.0
f 1 2 3
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := importers.NewVertexImporter()
			err := importer.ImportFromOBJ(strings.NewReader(tt.objData))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			vertices := importer.GetVertices()
			assert.Equal(t, tt.expected, vertices)
		})
	}
}

func TestImportFromFBX(t *testing.T) {
	// Create a minimal valid FBX binary file
	fbxData := createTestFBXBinary()

	tests := []struct {
		name    string
		fbxData []byte
		wantErr bool
	}{
		{
			name:    "valid FBX",
			fbxData: fbxData,
			wantErr: false,
		},
		{
			name:    "invalid magic number",
			fbxData: []byte("invalid FBX data"),
			wantErr: true,
		},
		{
			name:    "empty data",
			fbxData: []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := importers.NewVertexImporter()
			err := importer.ImportFromFBX(bytes.NewReader(tt.fbxData))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			vertices := importer.GetVertices()
			assert.NotEmpty(t, vertices)
		})
	}
}

// Helper function to create a test FBX binary file
func createTestFBXBinary() []byte {
	// FBX Binary format header
	header := []byte("Kaydara FBX Binary  ")
	header = append(header, []byte{0x00, 0x1A, 0x00}...)

	// FBX version (7400 = 0x1CE8)
	version := []byte{0xE8, 0x1C, 0x00, 0x00}

	// Simple geometry data with vertices
	geometryData := []byte{
		// Node header
		0x00, 0x00, 0x01, 0x00, // endOffset
		0x01, 0x00, 0x00, 0x00, // numProperties (1 array)
		0x48, 0x00, 0x00, 0x00, // propertyListLen (72 bytes - array header + data)
		0x08, // nameLen
		'V', 'e', 'r', 't', 'i', 'c', 'e', 's',

		// Property type (array of float64)
		'D', 0x00, // type code for float64 array
		0x09, 0x00, 0x00, 0x00, // array length (9 values - 3 vertices * 3 coordinates)
		0x00, 0x00, 0x00, 0x00, // encoding (0 = raw binary)
		0x00, 0x00, 0x00, 0x00, // compressed length (0 = uncompressed)

		// Vertex data (3 vertices as float64)
		// Vertex 1: (0, 0, 0)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Vertex 2: (1, 0, 0)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Vertex 3: (0, 1, 0)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// Combine all parts
	data := make([]byte, 0, len(header)+len(version)+len(geometryData))
	data = append(data, header...)
	data = append(data, version...)
	data = append(data, geometryData...)

	return data
}

func TestParseVector3(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    [3]float64
		wantErr bool
	}{
		{
			name:    "valid vector3",
			values:  []string{"1.0", "2.0", "3.0"},
			want:    [3]float64{1.0, 2.0, 3.0},
			wantErr: false,
		},
		{
			name:    "invalid number",
			values:  []string{"1.0", "invalid", "3.0"},
			wantErr: true,
		},
		{
			name:    "not enough values",
			values:  []string{"1.0", "2.0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := importers.ParseVector3(tt.values)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}
