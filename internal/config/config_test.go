package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/craigderington/prox/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "prox_config_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "prox.yml")
	configContent := `
services:
  web-server:
    script: server.js
    interpreter: node
    cwd: /app
    args:
      - --port
      - "3000"
    env:
      NODE_ENV: production
      PORT: "3000"
    restart: on-failure

  worker:
    script: worker.py
    interpreter: python3
    instances: 2
    env:
      PYTHONUNBUFFERED: "1"
`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	cfg, err := config.LoadConfig(configFile)
	require.NoError(t, err)

	// Verify services
	assert.Len(t, cfg.Services, 2)

	// Check web-server service
	webServer, exists := cfg.Services["web-server"]
	assert.True(t, exists)
	assert.Equal(t, "server.js", webServer.Script)
	assert.Equal(t, "node", webServer.Interpreter)
	assert.Equal(t, "/app", webServer.Cwd)
	assert.Equal(t, []string{"--port", "3000"}, webServer.Args)
	assert.Equal(t, "production", webServer.Env["NODE_ENV"])
	assert.Equal(t, "3000", webServer.Env["PORT"])
	assert.Equal(t, "on-failure", webServer.Restart)
	assert.Equal(t, 1, webServer.Instances) // Default value

	// Check worker service
	worker, exists := cfg.Services["worker"]
	assert.True(t, exists)
	assert.Equal(t, "worker.py", worker.Script)
	assert.Equal(t, "python3", worker.Interpreter)
	assert.Equal(t, 2, worker.Instances)
	assert.Equal(t, "1", worker.Env["PYTHONUNBUFFERED"])
	assert.Equal(t, "on-failure", worker.Restart) // Default value
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create temporary config file with invalid YAML
	tempDir, err := os.MkdirTemp("", "prox_config_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "prox.yml")
	invalidYAML := `
services:
  web-server:
    script: server.js
    invalid_yaml: [unclosed bracket
`

	err = os.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Try to load invalid config
	_, err = config.LoadConfig(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := config.LoadConfig("/non/existent/file.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestFindConfigFile(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "prox_find_config_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create a prox.yml file
	configFile := filepath.Join(tempDir, "prox.yml")
	err = os.WriteFile(configFile, []byte("services: {}"), 0644)
	require.NoError(t, err)

	// Test finding config file
	found, err := config.FindConfigFile()
	assert.NoError(t, err)
	assert.Equal(t, configFile, found)
}

func TestFindConfigFileAlternativeNames(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "prox_find_config_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Test different file names
	testCases := []string{"prox.yml", "prox.yaml", ".prox.yml"}

	for _, filename := range testCases {
		// Clean up previous files
		for _, name := range testCases {
			os.Remove(filepath.Join(tempDir, name))
		}

		// Create file with current name
		configFile := filepath.Join(tempDir, filename)
		err = os.WriteFile(configFile, []byte("services: {}"), 0644)
		require.NoError(t, err)

		// Test finding config file
		found, err := config.FindConfigFile()
		assert.NoError(t, err)
		assert.Equal(t, configFile, found)
	}
}

func TestFindConfigFileNotFound(t *testing.T) {
	// Create temporary directory with no config files
	tempDir, err := os.MkdirTemp("", "prox_find_config_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Test finding config file when none exists
	_, err = config.FindConfigFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no config file found")
}

func TestSaveConfig(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "prox_save_config_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "test.yml")

	// Create test config
	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			"api": {
				Script:      "server.js",
				Interpreter: "node",
				Args:        []string{"--port", "3000"},
				Env: map[string]string{
					"NODE_ENV": "production",
				},
				Restart: "on-failure",
			},
		},
	}

	// Save config
	err = config.SaveConfig(configFile, cfg)
	assert.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, configFile)

	// Load and verify content
	loaded, err := config.LoadConfig(configFile)
	assert.NoError(t, err)
	assert.Equal(t, cfg.Services["api"].Script, loaded.Services["api"].Script)
	assert.Equal(t, cfg.Services["api"].Interpreter, loaded.Services["api"].Interpreter)
	assert.Equal(t, cfg.Services["api"].Args, loaded.Services["api"].Args)
	assert.Equal(t, cfg.Services["api"].Env, loaded.Services["api"].Env)
	assert.Equal(t, cfg.Services["api"].Restart, loaded.Services["api"].Restart)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name:    "empty services",
			config:  &config.Config{Services: map[string]config.ServiceConfig{}},
			wantErr: true,
			errMsg:  "no services defined",
		},
		{
			name: "valid config",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {
						Script:      "server.js",
						Interpreter: "node",
						Instances:   1,
						Restart:     "on-failure",
						Env:         map[string]string{"NODE_ENV": "production"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing script",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {Interpreter: "node"},
				},
			},
			wantErr: true,
			errMsg:  "must have either 'script' or 'command'",
		},
		{
			name: "invalid service name",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web server": {Script: "server.js"},
				},
			},
			wantErr: true,
			errMsg:  "invalid service name",
		},
		{
			name: "invalid interpreter",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {
						Script:      "server.js",
						Interpreter: "invalid-interpreter",
					},
				},
			},
			wantErr: true,
			errMsg:  "unsupported interpreter",
		},
		{
			name: "invalid restart policy",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {
						Script:  "server.js",
						Restart: "invalid-policy",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid restart policy",
		},
		{
			name: "invalid env var",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {
						Script: "server.js",
						Env:    map[string]string{"invalid-var": "value"},
					},
				},
			},
			wantErr: true,
			errMsg:  "environment variable key",
		},
		{
			name: "negative instances",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {
						Script:    "server.js",
						Instances: -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "must have at least 1 instance",
		},
		{
			name: "dependency cycle",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"a": {Script: "a.js", Instances: 1, DependsOn: []string{"b"}},
					"b": {Script: "b.js", Instances: 1, DependsOn: []string{"a"}},
				},
			},
			wantErr: true,
			errMsg:  "dependency cycle",
		},
		{
			name: "unknown dependency",
			config: &config.Config{
				Services: map[string]config.ServiceConfig{
					"web": {
						Script:    "server.js",
						Instances: 1,
						DependsOn: []string{"unknown-service"},
					},
				},
			},
			wantErr: true,
			errMsg:  "depends on unknown service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
