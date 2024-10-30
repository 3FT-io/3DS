package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// CreateTempDir creates a temporary directory and returns its path along with a cleanup function
func CreateTempDir(t *testing.T, prefix string) (string, func()) {
	tmpDir, err := os.MkdirTemp("", prefix)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// CreateTestFile creates a temporary file with the given content and returns its path
func CreateTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}
