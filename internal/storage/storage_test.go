package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/craigderington/prox/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) *storage.Storage {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "prox_storage_test_*")
	require.NoError(t, err)

	// Change to temp directory so storage uses it
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	t.Cleanup(func() {
		os.Chdir(oldWd)
		os.RemoveAll(tempDir)
	})

	storage, err := storage.New()
	require.NoError(t, err)
	return storage
}

func TestStorage_New(t *testing.T) {
	storage := setupTestStorage(t)

	// Verify config directory exists
	assert.DirExists(t, storage.ConfigDir())

	// Verify subdirectories exist
	assert.DirExists(t, storage.ProcessesDir())
	assert.DirExists(t, storage.LogsDir())
	assert.DirExists(t, storage.PidsDir())
}

func TestStorage_SaveLoadState(t *testing.T) {
	storage := setupTestStorage(t)

	// Test data
	testData := map[string]interface{}{
		"version":   "1.0.0",
		"processes": []string{"app1", "app2"},
		"settings": map[string]string{
			"auto_restart": "true",
		},
	}

	// Save state
	err := storage.SaveState(testData)
	assert.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, storage.StateFile())

	// Load state
	var loadedData map[string]interface{}
	err = storage.LoadState(&loadedData)
	assert.NoError(t, err)

	// Verify data (JSON unmarshaling changes types)
	assert.Equal(t, "1.0.0", loadedData["version"])

	processes, ok := loadedData["processes"].([]interface{})
	require.True(t, ok)
	assert.Len(t, processes, 2)
	assert.Equal(t, "app1", processes[0])
	assert.Equal(t, "app2", processes[1])

	settings, ok := loadedData["settings"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "true", settings["auto_restart"])
}

func TestStorage_SaveLoadStateNonExistent(t *testing.T) {
	storage := setupTestStorage(t)

	// Try to load from non-existent file
	var data map[string]interface{}
	err := storage.LoadState(&data)
	assert.NoError(t, err) // Should not error for missing file
}

func TestStorage_SavePID(t *testing.T) {
	storage := setupTestStorage(t)

	// Save PID
	err := storage.SavePID("test-app", 12345)
	assert.NoError(t, err)

	// Verify file exists and contains correct PID
	pidFile := filepath.Join(storage.PidsDir(), "test-app.pid")
	assert.FileExists(t, pidFile)

	content, err := os.ReadFile(pidFile)
	assert.NoError(t, err)
	assert.Equal(t, "12345", string(content))
}

func TestStorage_RemovePID(t *testing.T) {
	storage := setupTestStorage(t)

	// Save PID first
	err := storage.SavePID("test-app", 12345)
	require.NoError(t, err)

	pidFile := filepath.Join(storage.PidsDir(), "test-app.pid")
	assert.FileExists(t, pidFile)

	// Remove PID
	err = storage.RemovePID("test-app")
	assert.NoError(t, err)

	// Verify file is gone
	assert.NoFileExists(t, pidFile)
}

func TestStorage_RemovePIDNonExistent(t *testing.T) {
	storage := setupTestStorage(t)

	// Try to remove non-existent PID file
	err := storage.RemovePID("non-existent")
	assert.NoError(t, err) // Should not error for missing file
}

func TestStorage_GetLogFile(t *testing.T) {
	storage := setupTestStorage(t)

	// Test stdout log file path
	outLog := storage.GetLogFile("my-app", "out")
	expectedOut := filepath.Join(storage.LogsDir(), "my-app-out.log")
	assert.Equal(t, expectedOut, outLog)

	// Test stderr log file path
	errLog := storage.GetLogFile("my-app", "err")
	expectedErr := filepath.Join(storage.LogsDir(), "my-app-err.log")
	assert.Equal(t, expectedErr, errLog)
}
