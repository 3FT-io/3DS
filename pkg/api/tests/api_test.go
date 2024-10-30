package api_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/3FT-io/3DS/pkg/api"
	"github.com/3FT-io/3DS/pkg/core"
	"github.com/3FT-io/3DS/pkg/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Add APIResponse type or use the qualified name
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func setupTestAPI(t *testing.T) (*api.API, func()) {
	// Create temporary directory for storage
	tmpDir, err := os.MkdirTemp("", "3ds-test-*")
	require.NoError(t, err)

	// Initialize components
	storage, err := core.NewStorage(tmpDir)
	require.NoError(t, err)

	node := &core.Node{}
	network := &p2p.Network{}

	// Create API instance
	apiInstance, err := api.NewAPI(node, network, storage, 0)
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return apiInstance, cleanup
}

func TestHealthCheck(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call health check handler
	api.HealthCheck(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestUploadModel(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test file
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	fileWriter, err := writer.CreateFormFile("model", "test.gltf")
	require.NoError(t, err)

	_, err = fileWriter.Write([]byte("test model content"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Create test request
	req := httptest.NewRequest("POST", "/models", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Call upload handler
	api.UploadModel(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.True(t, response.Success)
}
