package importers_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/3FT-io/3DS/pkg/importers"
)

func TestImportFromMTL(t *testing.T) {
	tests := []struct {
		name     string
		mtlData  string
		expected map[string]*importers.Material
		wantErr  bool
	}{
		{
			name: "simple material",
			mtlData: `
newmtl Material1
Ka 0.1 0.1 0.1
Kd 0.8 0.8 0.8
Ks 1.0 1.0 1.0
Ns 50.0
d 1.0
map_Kd diffuse.png
map_Bump normal.png
map_Ks specular.png
`,
			expected: map[string]*importers.Material{
				"Material1": {
					Name:          "Material1",
					AmbientColor:  [3]float64{0.1, 0.1, 0.1},
					DiffuseColor:  [3]float64{0.8, 0.8, 0.8},
					SpecularColor: [3]float64{1.0, 1.0, 1.0},
					Shininess:     50.0,
					Transparency:  1.0,
					DiffuseMap:    "diffuse.png",
					NormalMap:     "normal.png",
					SpecularMap:   "specular.png",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple materials",
			mtlData: `
newmtl Material1
Ka 0.1 0.1 0.1
Kd 0.8 0.8 0.8

newmtl Material2
Ka 0.2 0.2 0.2
Kd 0.9 0.9 0.9
`,
			expected: map[string]*importers.Material{
				"Material1": {
					Name:         "Material1",
					AmbientColor: [3]float64{0.1, 0.1, 0.1},
					DiffuseColor: [3]float64{0.8, 0.8, 0.8},
				},
				"Material2": {
					Name:         "Material2",
					AmbientColor: [3]float64{0.2, 0.2, 0.2},
					DiffuseColor: [3]float64{0.9, 0.9, 0.9},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid color value",
			mtlData: `
newmtl Material1
Ka 0.1 invalid 0.1
`,
			wantErr: true,
		},
		{
			name: "property before material",
			mtlData: `
Ka 0.1 0.1 0.1
newmtl Material1
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := importers.NewMaterialImporter()
			err := importer.ImportFromOBJ(strings.NewReader(tt.mtlData))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			materials := importer.GetMaterials()
			assert.Equal(t, tt.expected, materials)
		})
	}
}

func TestImportFromFBXMaterial(t *testing.T) {
	// Create test FBX data with materials
	fbxData := createTestFBXMaterial()

	tests := []struct {
		name    string
		fbxData []byte
		wantErr bool
	}{
		{
			name:    "valid FBX material",
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
			importer := importers.NewMaterialImporter()
			err := importer.ImportFromFBX(bytes.NewReader(tt.fbxData))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			materials := importer.GetMaterials()
			assert.NotEmpty(t, materials)
		})
	}
}

func createTestFBXMaterial() []byte {
	// Reuse the FBX header and version from vertex test
	header := []byte("Kaydara FBX Binary  ")
	header = append(header, []byte{0x00, 0x1A, 0x00}...)
	version := []byte{0xE8, 0x1C, 0x00, 0x00}

	// Create material node data
	materialData := []byte{
		// Node header
		0x00, 0x00, 0x02, 0x00, // endOffset
		0x00, 0x00, 0x00, 0x00, // numProperties
		0x00, 0x00, 0x00, 0x00, // propertyListLen
		0x08, // nameLen
		'M', 'a', 't', 'e', 'r', 'i', 'a', 'l',
	}

	// Combine all parts
	data := make([]byte, 0, len(header)+len(version)+len(materialData))
	data = append(data, header...)
	data = append(data, version...)
	data = append(data, materialData...)

	return data
}
