package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the prox.yml configuration
type Config struct {
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig represents a single service configuration
type ServiceConfig struct {
	Command     string            `yaml:"command"`
	Script      string            `yaml:"script,omitempty"`
	Interpreter string            `yaml:"interpreter,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Cwd         string            `yaml:"cwd,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Instances   int               `yaml:"instances,omitempty"`
	Restart     string            `yaml:"restart,omitempty"` // "always", "on-failure", "never"
	Watch       []string          `yaml:"watch,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	HealthCheck *HealthCheck      `yaml:"healthcheck,omitempty"`
}

// HealthCheck configuration
type HealthCheck struct {
	HTTPGet  string `yaml:"http_get,omitempty"`
	Interval string `yaml:"interval,omitempty"`
	Timeout  string `yaml:"timeout,omitempty"`
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults
	for name, svc := range config.Services {
		if svc.Instances == 0 {
			svc.Instances = 1
		}
		if svc.Restart == "" {
			svc.Restart = "on-failure"
		}
		config.Services[name] = svc
	}

	return &config, nil
}

// FindConfigFile looks for prox.yml in current directory and parent directories
func FindConfigFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Try common config file names
	names := []string{"prox.yml", "prox.yaml", ".prox.yml"}

	dir := cwd
	for {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no config file found (looked for prox.yml)")
}

// SaveConfig saves configuration to a file
func SaveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
