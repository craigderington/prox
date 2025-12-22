package cmd

import (
	"github.com/craigderington/prox/internal/manager"
	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
)

// getManager returns the shared process manager and storage
func getManager() (*process.Manager, *storage.Storage, error) {
	return manager.Get()
}

// saveState saves the current manager state
func saveState() error {
	return manager.Save()
}
