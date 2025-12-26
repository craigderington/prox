package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Config represents the prox.yml configuration file structure.
// It contains all the services to be managed by prox.
type Config struct {
	// Services is a map of service names to their configurations
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig represents a single service configuration.
// Each service defines how a process should be started and managed.
type ServiceConfig struct {
	// Command is the full command to execute (alternative to Script+Interpreter)
	Command string `yaml:"command,omitempty"`
	// Script is the script file to execute
	Script string `yaml:"script,omitempty"`
	// Interpreter is the interpreter to use (node, python, ruby, etc.)
	Interpreter string `yaml:"interpreter,omitempty"`
	// Args are additional command line arguments
	Args []string `yaml:"args,omitempty"`
	// Cwd is the working directory for the process
	Cwd string `yaml:"cwd,omitempty"`
	// Env is environment variables to set
	Env map[string]string `yaml:"env,omitempty"`
	// Instances is the number of instances to run (default: 1)
	Instances int `yaml:"instances,omitempty"`
	// Restart defines when to restart the process ("always", "on-failure", "never")
	Restart string `yaml:"restart,omitempty"`
	// Watch is a list of files/directories to watch for changes
	Watch []string `yaml:"watch,omitempty"`
	// DependsOn is a list of services this service depends on
	DependsOn []string `yaml:"depends_on,omitempty"`
	// HealthCheck defines how to check if the service is healthy
	HealthCheck *HealthCheck `yaml:"healthcheck,omitempty"`
}

// HealthCheck defines health check configuration for a service.
type HealthCheck struct {
	// HTTPGet is the HTTP endpoint to check
	HTTPGet string `yaml:"http_get,omitempty"`
	// Interval is how often to perform the health check
	Interval string `yaml:"interval,omitempty"`
	// Timeout is how long to wait for the health check to complete
	Timeout string `yaml:"timeout,omitempty"`
}

// LoadConfig loads and validates configuration from a YAML file.
// It parses the prox.yml file, validates the configuration, and applies defaults.
// Returns an error if the file cannot be read, parsed, or if validation fails.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate configuration before setting defaults
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set defaults after validation
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

// ValidateConfig performs comprehensive validation of the configuration.
// It checks for required fields, valid values, dependency cycles, and other
// configuration constraints. Returns nil if the configuration is valid.
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if len(config.Services) == 0 {
		return fmt.Errorf("no services defined in configuration")
	}

	// Track service names for uniqueness and dependency validation
	serviceNames := make(map[string]bool)
	dependencyGraph := make(map[string][]string)

	for name, service := range config.Services {
		// Check for duplicate service names
		if serviceNames[name] {
			return fmt.Errorf("duplicate service name: %s", name)
		}
		serviceNames[name] = true

		// Validate service name format
		if err := validateServiceName(name); err != nil {
			return fmt.Errorf("invalid service name '%s': %w", name, err)
		}

		// Validate required fields
		if service.Script == "" && service.Command == "" {
			return fmt.Errorf("service '%s' must have either 'script' or 'command' defined", name)
		}

		// Validate interpreter
		if service.Interpreter != "" {
			if err := validateInterpreter(service.Interpreter); err != nil {
				return fmt.Errorf("service '%s' has invalid interpreter '%s': %w", name, service.Interpreter, err)
			}
		}

		// Validate restart policy
		if err := validateRestartPolicy(service.Restart); err != nil {
			return fmt.Errorf("service '%s' has invalid restart policy '%s': %w", name, service.Restart, err)
		}

		// Validate instances (only if explicitly set, since defaults will be applied later)
		if service.Instances != 0 && service.Instances < 1 {
			return fmt.Errorf("service '%s' must have at least 1 instance (got %d)", name, service.Instances)
		}

		// Validate environment variables
		for key, value := range service.Env {
			if err := validateEnvVar(key, value); err != nil {
				return fmt.Errorf("service '%s' has invalid environment variable %s=%s: %w", name, key, value, err)
			}
		}

		// Build dependency graph
		dependencyGraph[name] = service.DependsOn
	}

	// Check for dependency cycles
	if err := detectDependencyCycles(dependencyGraph); err != nil {
		return fmt.Errorf("dependency cycle detected: %w", err)
	}

	// Validate that all dependencies exist
	for serviceName, deps := range dependencyGraph {
		for _, dep := range deps {
			if !serviceNames[dep] {
				return fmt.Errorf("service '%s' depends on unknown service '%s'", serviceName, dep)
			}
		}
	}

	return nil
}

// validateServiceName validates that a service name follows proper conventions
func validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if len(name) > 64 {
		return fmt.Errorf("service name too long (max 64 characters)")
	}

	// Service names should be valid identifiers (alphanumeric, dash, underscore)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("service name must contain only letters, numbers, dashes, and underscores")
	}

	return nil
}

// validateInterpreter validates that an interpreter is supported
func validateInterpreter(interpreter string) error {
	supported := map[string]bool{
		"node":    true,
		"nodejs":  true,
		"python":  true,
		"python2": true,
		"python3": true,
		"ruby":    true,
		"perl":    true,
		"php":     true,
		"bash":    true,
		"sh":      true,
		"zsh":     true,
		"fish":    true,
		"go":      true,
		"rust":    true,
		"java":    true,
	}

	if !supported[strings.ToLower(interpreter)] {
		return fmt.Errorf("unsupported interpreter (supported: node, python, ruby, perl, php, bash, sh, zsh, fish, go, rust, java)")
	}

	return nil
}

// validateRestartPolicy validates restart policy values
func validateRestartPolicy(policy string) error {
	validPolicies := map[string]bool{
		"always":     true,
		"on-failure": true,
		"never":      true,
		"":           true, // Empty is allowed (will be set to default)
	}

	if !validPolicies[strings.ToLower(policy)] {
		return fmt.Errorf("invalid restart policy (must be: always, on-failure, never)")
	}

	return nil
}

// validateEnvVar validates environment variable format
func validateEnvVar(key, value string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	// Environment variable names should be valid
	validKey := regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	if !validKey.MatchString(key) {
		return fmt.Errorf("environment variable key must start with letter/underscore and contain only uppercase letters, numbers, and underscores")
	}

	return nil
}

// detectDependencyCycles detects circular dependencies in the service dependency graph
func detectDependencyCycles(graph map[string][]string) error {
	// Simple cycle detection using DFS
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var dfs func(string) error
	dfs = func(service string) error {
		visited[service] = true
		recursionStack[service] = true

		for _, dep := range graph[service] {
			if !visited[dep] {
				if err := dfs(dep); err != nil {
					return err
				}
			} else if recursionStack[dep] {
				return fmt.Errorf("cycle involving %s -> %s", service, dep)
			}
		}

		recursionStack[service] = false
		return nil
	}

	for service := range graph {
		if !visited[service] {
			if err := dfs(service); err != nil {
				return err
			}
		}
	}

	return nil
}

// FindConfigFile searches for a prox configuration file.
// It looks for prox.yml, prox.yaml, or .prox.yml in the current directory
// and parent directories, returning the path to the first one found.
// Returns an error if no configuration file is found.
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

// SaveConfig writes configuration to a YAML file.
// It serializes the config struct to YAML format and writes it to the specified path.
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
