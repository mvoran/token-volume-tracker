package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configFileName = "config.json"
)

// Config represents the application configuration
type Config struct {
	CMCApiKey string `json:"cmc_api_key"`
}

// DefaultConfigPath returns the default path for the config file
func DefaultConfigPath() (string, error) {
	// Get the executable's directory
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Get the working directory
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting working directory: %w", err)
	}

	// If we're in development mode (running with 'go run'), use the working directory
	// Otherwise, use the executable's directory
	if filepath.Base(execDir) == "go-build" {
		return filepath.Join(workDir, configFileName), nil
	}
	return filepath.Join(execDir, configFileName), nil
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{}, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
