package process_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
	"github.com/stretchr/testify/require"
)

// setupBenchmarkStorage creates storage for benchmarks
func setupBenchmarkStorage(b *testing.B) *storage.Storage {
	// Create temporary directory for benchmarks
	tempDir, err := os.MkdirTemp("", "prox_bench_*")
	require.NoError(b, err)

	// Change to temp directory so storage uses it
	oldWd, _ := os.Getwd()
	b.Cleanup(func() {
		os.Chdir(oldWd)
		os.RemoveAll(tempDir)
	})

	storage, err := storage.New()
	require.NoError(b, err)
	return storage
}

// BenchmarkMetricsCollector_CollectMetrics benchmarks the metrics collection
func BenchmarkMetricsCollector_CollectMetrics(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	manager := process.NewManager()
	manager.SetStorage(storage)
	collector := process.NewMetricsCollector(manager)

	// Start a test process
	config := process.ProcessConfig{
		Name:   "bench-process",
		Script: "sleep",
		Args:   []string{"10"},
	}

	proc, err := manager.Start(config)
	require.NoError(b, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectMetrics(proc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMetricsCollector_CollectAllMetrics benchmarks collecting metrics for all processes
func BenchmarkMetricsCollector_CollectAllMetrics(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	manager := process.NewManager()
	manager.SetStorage(storage)
	collector := process.NewMetricsCollector(manager)

	// Start multiple test processes
	processes := []process.ProcessConfig{
		{Name: "bench-1", Script: "sleep", Args: []string{"10"}},
		{Name: "bench-2", Script: "sleep", Args: []string{"10"}},
		{Name: "bench-3", Script: "sleep", Args: []string{"10"}},
	}

	for _, config := range processes {
		_, err := manager.Start(config)
		require.NoError(b, err)
	}

	// Give them a moment to start
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := collector.CollectAllMetrics()
		if len(metrics) == 0 {
			b.Fatal("Expected metrics for running processes")
		}
	}
}

// BenchmarkManager_List benchmarks the process listing operation
func BenchmarkManager_List(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	manager := process.NewManager()
	manager.SetStorage(storage)

	// Start multiple test processes
	for i := 0; i < 10; i++ {
		config := process.ProcessConfig{
			Name:   fmt.Sprintf("bench-%d", i),
			Script: "sleep",
			Args:   []string{"10"},
		}
		_, err := manager.Start(config)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := manager.List()
		if len(list) != 10 {
			b.Fatalf("Expected 10 processes, got %d", len(list))
		}
	}
}
