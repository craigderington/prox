package manager

import (
	"fmt"
	"sync"

	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
)

var (
	instance     *process.Manager
	storageInst  *storage.Storage
	instanceOnce sync.Once
	instanceErr  error
)

// Get returns the shared process manager instance
func Get() (*process.Manager, *storage.Storage, error) {
	instanceOnce.Do(func() {
		// Initialize storage
		storageInst, instanceErr = storage.New()
		if instanceErr != nil {
			instanceErr = fmt.Errorf("failed to initialize storage: %w", instanceErr)
			return
		}

		// Create manager
		instance = process.NewManager()

		// Set storage on manager for logging
		instance.SetStorage(storageInst)

		// Load state from disk
		if err := instance.LoadState(storageInst); err != nil {
			instanceErr = fmt.Errorf("failed to load state: %w", err)
			return
		}

		// Restore monitoring for running processes
		if err := instance.RestoreRunningProcesses(); err != nil {
			instanceErr = fmt.Errorf("failed to restore processes: %w", err)
			return
		}
	})

	return instance, storageInst, instanceErr
}

// Save saves the current manager state
func Save() error {
	if instance == nil || storageInst == nil {
		return fmt.Errorf("manager not initialized")
	}
	return instance.SaveState(storageInst)
}
