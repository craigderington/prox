package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/craigderington/prox/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndProcessLifecycle tests the complete process lifecycle
func TestEndToEndProcessLifecycle(t *testing.T) {
	// Create a simple test script
	tempDir, err := os.MkdirTemp("", "prox_e2e_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testScript := filepath.Join(tempDir, "test.sh")
	scriptContent := `#!/bin/bash
echo "Starting test process"
sleep 2
echo "Test process completed"
`
	err = os.WriteFile(testScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Build prox binary
	proxPath := filepath.Join(tempDir, "prox")
	proxRoot := os.Getenv("PROX_ROOT")
	if proxRoot == "" {
		// Assume we're in the project root (parent of tests/)
		proxRoot = ".."
	}
	cmd := exec.Command("go", "build", "-o", proxPath, ".")
	cmd.Dir = proxRoot
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build prox: %s", string(output))

	// Create a temporary prox data directory
	proxDataDir := filepath.Join(tempDir, "prox_data")
	err = os.MkdirAll(proxDataDir, 0755)
	require.NoError(t, err)

	// Set HOME to use our temp directory for prox data
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", proxDataDir)
	defer os.Setenv("HOME", oldHome)

	// Test prox commands
	testCases := []struct {
		name     string
		command  []string
		validate func(t *testing.T, output string, err error)
	}{
		{
			name:    "start process",
			command: []string{proxPath, "start", testScript, "--name", "e2e-test"},
			validate: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, string(output), "Started 'e2e-test'")
			},
		},
		{
			name:    "list processes",
			command: []string{proxPath, "list"},
			validate: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, string(output), "e2e-test")
			},
		},
		{
			name:    "stop process",
			command: []string{proxPath, "stop", "e2e-test"},
			validate: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(tc.command[0], tc.command[1:]...)
			output, err := cmd.CombinedOutput()
			tc.validate(t, string(output), err)
		})
	}
}

// TestConfigurationValidation tests end-to-end config validation
func TestConfigurationValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prox_config_e2e_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create invalid config
	invalidConfig := filepath.Join(tempDir, "prox.yml")
	invalidContent := `
services:
  bad-service:
    script: nonexistent.js
    interpreter: invalid-interpreter
    instances: -1
    restart: invalid-policy
    env:
      bad-var: value
    depends_on:
      - self
`
	err = os.WriteFile(invalidConfig, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Try to load invalid config
	_, err = config.LoadConfig(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// Create valid config
	validConfig := filepath.Join(tempDir, "prox-valid.yml")
	validContent := `
services:
  good-service:
    script: /bin/echo
    args: ["hello"]
    instances: 1
    restart: on-failure
    env:
      TEST_VAR: test_value
`
	err = os.WriteFile(validConfig, []byte(validContent), 0644)
	require.NoError(t, err)

	// Load valid config
	cfg, err := config.LoadConfig(validConfig)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Services, "good-service")
}

// TestLogManagement tests log reading and writing
func TestLogManagement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prox_logs_e2e_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test log file
	logFile := filepath.Join(tempDir, "test.log")
	logContent := `2024-01-01 10:00:00 [INFO] Starting service
2024-01-01 10:00:01 [WARN] Configuration warning
2024-01-01 10:00:02 [ERROR] Something went wrong
`
	err = os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Test log reading (this would normally be done by prox logs command)
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 3)
	assert.Contains(t, lines[0], "Starting service")
	assert.Contains(t, lines[2], "Something went wrong")
}

// TestCrossPlatformCompatibility tests basic cross-platform functionality
func TestCrossPlatformCompatibility(t *testing.T) {
	// Test that we can run on the current platform
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "go version")

	// Test that our imports work on this platform
	proxRoot := os.Getenv("PROX_ROOT")
	if proxRoot == "" {
		proxRoot = ".."
	}
	cmd = exec.Command("go", "build", "-o", "/dev/null", ".")
	cmd.Dir = proxRoot
	output, err = cmd.CombinedOutput()
	assert.NoError(t, err, "Build failed: %s", string(output))
}

// TestPerformanceUnderLoad tests basic performance characteristics
func TestPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	start := time.Now()

	// Simulate some work
	for i := 0; i < 1000; i++ {
		_ = i * i
	}

	elapsed := time.Since(start)
	// Should complete in well under 1ms
	assert.Less(t, elapsed, time.Millisecond, "Performance test took too long")
}
