package process_test

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) *storage.Storage {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "prox_test_*")
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

func TestManager_Start(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Test starting a simple process
	config := process.ProcessConfig{
		Name:        "test-app",
		Script:      "echo",
		Args:        []string{"hello"},
		Interpreter: "",
	}

	proc, err := manager.Start(config)
	require.NoError(t, err)
	assert.Equal(t, "test-app", proc.Name)
	assert.Equal(t, "echo", proc.Script)
	assert.Equal(t, process.StatusOnline, proc.Status)
	assert.Greater(t, proc.PID, 0)

	// Verify process exists in manager
	found := manager.Get("test-app")
	assert.NotNil(t, found)
	assert.Equal(t, proc.ID, found.ID)
}

func TestManager_StartDuplicateName(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	config := process.ProcessConfig{
		Name:   "test-app",
		Script: "echo",
	}

	// Start first process
	_, err := manager.Start(config)
	require.NoError(t, err)

	// Try to start second process with same name
	_, err = manager.Start(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_Stop(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Start a long-running process
	config := process.ProcessConfig{
		Name:   "test-app",
		Script: "sleep",
		Args:   []string{"5"},
	}

	proc, err := manager.Start(config)
	require.NoError(t, err)

	// Stop the process
	err = manager.Stop("test-app")
	assert.NoError(t, err)

	// Verify process is stopped
	found := manager.Get("test-app")
	assert.Equal(t, process.StatusStopped, found.Status)
	assert.Equal(t, proc.ID, found.ID) // ensure we use proc
}

func TestManager_List(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Start multiple processes
	processes := []process.ProcessConfig{
		{Name: "app1", Script: "echo", Args: []string{"1"}},
		{Name: "app2", Script: "echo", Args: []string{"2"}},
		{Name: "app3", Script: "echo", Args: []string{"3"}},
	}

	for _, config := range processes {
		_, err := manager.Start(config)
		require.NoError(t, err)
	}

	// List all processes
	list := manager.List()
	assert.Len(t, list, 3)

	// Verify alphabetical ordering
	assert.Equal(t, "app1", list[0].Name)
	assert.Equal(t, "app2", list[1].Name)
	assert.Equal(t, "app3", list[2].Name)
}

func TestManager_Restart(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Start a process
	config := process.ProcessConfig{
		Name:   "test-app",
		Script: "sleep",
		Args:   []string{"10"},
	}

	proc, err := manager.Start(config)
	require.NoError(t, err)
	originalPID := proc.PID

	// Restart the process
	err = manager.Restart("test-app")
	assert.NoError(t, err)

	// Verify process restarted with new PID
	found := manager.Get("test-app")
	assert.Equal(t, process.StatusOnline, found.Status)
	assert.NotEqual(t, originalPID, found.PID) // Should have new PID
}

func TestManager_Delete(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Start a process
	config := process.ProcessConfig{
		Name:   "test-app",
		Script: "sleep",
		Args:   []string{"10"},
	}

	_, err := manager.Start(config)
	require.NoError(t, err)

	// Delete the process
	err = manager.Delete("test-app")
	assert.NoError(t, err)

	// Verify process is gone
	found := manager.Get("test-app")
	assert.Nil(t, found)

	// Verify not in list
	list := manager.List()
	assert.Len(t, list, 0)
}

func TestManager_ConcurrentAccess(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Test concurrent starts and stops
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			config := process.ProcessConfig{
				Name:   fmt.Sprintf("app-%d", id),
				Script: "sleep",
				Args:   []string{"1"}, // Sleep for 1 second
			}

			// Start process
			_, err := manager.Start(config)
			assert.NoError(t, err)

			// Wait a bit for process to start
			time.Sleep(50 * time.Millisecond)

			// Stop process
			err = manager.Stop(config.Name)
			// Don't assert NoError here as the process might have already exited
			// The important thing is that Stop() doesn't panic
			_ = err
		}(i)
	}

	wg.Wait()

	// Give processes time to fully stop
	time.Sleep(100 * time.Millisecond)

	// All processes should be stopped or stopping
	list := manager.List()
	for _, proc := range list {
		assert.True(t, proc.Status == process.StatusStopped ||
			proc.Status == process.StatusStopping,
			"Process %s should be stopped or stopping, got %s", proc.Name, proc.Status)
	}
}

func TestManager_GetByID(t *testing.T) {
	storage := setupTestStorage(t)
	manager := process.NewManager()
	manager.SetStorage(storage)

	config := process.ProcessConfig{
		Name:   "test-app",
		Script: "echo",
	}

	proc, err := manager.Start(config)
	require.NoError(t, err)

	// Get by ID
	found := manager.Get(proc.ID)
	assert.NotNil(t, found)
	assert.Equal(t, proc.ID, found.ID)

	// Get by name
	found = manager.Get("test-app")
	assert.NotNil(t, found)
	assert.Equal(t, proc.ID, found.ID)

	// Get non-existent
	found = manager.Get("non-existent")
	assert.Nil(t, found)
}
