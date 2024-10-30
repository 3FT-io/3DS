package core_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/3FT-io/3DS/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) (*core.Storage, func()) {
	tmpDir, err := os.MkdirTemp("", "3ds-storage-test-*")
	require.NoError(t, err)

	storage, err := core.NewStorage(tmpDir)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return storage, cleanup
}

func TestStoreAndRetrieveModel(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	modelContent := "test model content"
	reader := strings.NewReader(modelContent)

	// Store model
	metadata, err := storage.StoreModel(ctx, "test.gltf", "gltf", reader)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.ID)

	// Retrieve model
	retrievedMetadata, err := storage.GetModel(ctx, metadata.ID)
	require.NoError(t, err)
	assert.Equal(t, metadata.ID, retrievedMetadata.ID)
	assert.Equal(t, "test.gltf", retrievedMetadata.Name)
}

func TestListModels(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Store multiple models and keep track of their IDs
	var storedIDs []string
	for i := 0; i < 3; i++ {
		reader := strings.NewReader("test content")
		metadata, err := storage.StoreModel(ctx, "test.gltf", "gltf", reader)
		require.NoError(t, err)
		storedIDs = append(storedIDs, metadata.ID)

		// Verify the model was stored
		_, err = storage.GetModel(ctx, metadata.ID)
		require.NoError(t, err)
	}

	// List models
	models, err := storage.ListModels(ctx)
	require.NoError(t, err)

	// Debug output
	t.Logf("Stored IDs: %v", storedIDs)
	t.Logf("Listed models count: %d", len(models))

	assert.Len(t, models, 3)

	// Verify each stored ID is present in the listed models
	for _, id := range storedIDs {
		found := false
		for _, model := range models {
			if model.ID == id {
				found = true
				break
			}
		}
		assert.True(t, found, "Model with ID %s not found in listed models", id)
	}
}

func TestDeleteModel(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	reader := strings.NewReader("test content")

	// Store model
	metadata, err := storage.StoreModel(ctx, "test.gltf", "gltf", reader)
	require.NoError(t, err)

	// Delete model
	err = storage.DeleteModel(ctx, metadata.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = storage.GetModel(ctx, metadata.ID)
	assert.Error(t, err)
}
