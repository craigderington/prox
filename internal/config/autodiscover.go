package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AutoDiscover attempts to discover services from common configuration files
func AutoDiscover() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Try discovery methods in order
	discoverers := []func(string) (*Config, error){
		discoverFromProcfile,
		discoverFromPackageJSON,
		discoverFromDockerCompose,
	}

	for _, discover := range discoverers {
		if config, err := discover(cwd); err == nil && config != nil {
			return config, nil
		}
	}

	return nil, fmt.Errorf("no supported configuration files found")
}

// AutoDiscoverFromFile discovers services from a specific file path
func AutoDiscoverFromFile(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	// Determine file type by name/extension
	filename := filepath.Base(path)
	dir := filepath.Dir(path)
	if dir == "." {
		dir, _ = os.Getwd()
	}

	// Try different parsers based on filename
	if filename == "Procfile" || strings.HasPrefix(filename, "Procfile.") {
		return parseSpecificProcfile(path, dir)
	} else if filename == "package.json" {
		return discoverFromPackageJSON(dir)
	} else if strings.Contains(filename, "docker-compose") {
		return parseDockerCompose(path, dir)
	}

	return nil, fmt.Errorf("unsupported file type: %s (expected Procfile, package.json, or docker-compose.yml)", filename)
}

// parseSpecificProcfile reads a Procfile from a specific path
func parseSpecificProcfile(path, baseDir string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Services: make(map[string]ServiceConfig),
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		command := strings.TrimSpace(parts[1])

		config.Services[name] = ServiceConfig{
			Command:   command,
			Restart:   "on-failure",
			Cwd:       baseDir,
			Instances: 1,
		}
	}

	if len(config.Services) == 0 {
		return nil, fmt.Errorf("no services found in Procfile")
	}

	return config, nil
}

// discoverFromProcfile reads Procfile format
func discoverFromProcfile(dir string) (*Config, error) {
	path := filepath.Join(dir, "Procfile")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Services: make(map[string]ServiceConfig),
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		command := strings.TrimSpace(parts[1])

		config.Services[name] = ServiceConfig{
			Command:  command,
			Restart:  "on-failure",
			Cwd:      dir,
			Instances: 1,
		}
	}

	if len(config.Services) == 0 {
		return nil, fmt.Errorf("no services found in Procfile")
	}

	return config, nil
}

// discoverFromPackageJSON reads package.json scripts
func discoverFromPackageJSON(dir string) (*Config, error) {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	if len(pkg.Scripts) == 0 {
		return nil, fmt.Errorf("no scripts found in package.json")
	}

	config := &Config{
		Services: make(map[string]ServiceConfig),
	}

	// Look for common development scripts
	devScripts := []string{"dev", "start", "serve"}
	for _, scriptName := range devScripts {
		if _, ok := pkg.Scripts[scriptName]; ok {
			config.Services[scriptName] = ServiceConfig{
				Command:   "npm run " + scriptName,
				Restart:   "on-failure",
				Cwd:       dir,
				Instances: 1,
			}
		}
	}

	if len(config.Services) == 0 {
		return nil, fmt.Errorf("no dev scripts found in package.json")
	}

	return config, nil
}

// discoverFromDockerCompose reads docker-compose.yml
func discoverFromDockerCompose(dir string) (*Config, error) {
	paths := []string{
		filepath.Join(dir, "docker-compose.yml"),
		filepath.Join(dir, "docker-compose.yaml"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return parseDockerCompose(path, dir)
		}
	}

	return nil, fmt.Errorf("no docker-compose file found")
}

// parseDockerCompose parses docker-compose.yml (simplified)
func parseDockerCompose(path, dir string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var compose struct {
		Services map[string]struct {
			Command   string   `yaml:"command"`
			DependsOn []string `yaml:"depends_on"`
		} `yaml:"services"`
	}

	if err := json.Unmarshal(data, &compose); err != nil {
		return nil, err
	}

	config := &Config{
		Services: make(map[string]ServiceConfig),
	}

	for name, svc := range compose.Services {
		if svc.Command != "" {
			config.Services[name] = ServiceConfig{
				Command:   svc.Command,
				Restart:   "on-failure",
				Cwd:       dir,
				Instances: 1,
				DependsOn: svc.DependsOn,
			}
		}
	}

	if len(config.Services) == 0 {
		return nil, fmt.Errorf("no runnable services found in docker-compose")
	}

	return config, nil
}

// GetDiscoverySource returns what file was used for discovery
func GetDiscoverySource() string {
	cwd, _ := os.Getwd()

	if _, err := os.Stat(filepath.Join(cwd, "Procfile")); err == nil {
		return "Procfile"
	}
	if _, err := os.Stat(filepath.Join(cwd, "package.json")); err == nil {
		return "package.json"
	}
	if _, err := os.Stat(filepath.Join(cwd, "docker-compose.yml")); err == nil {
		return "docker-compose.yml"
	}

	return "unknown"
}
