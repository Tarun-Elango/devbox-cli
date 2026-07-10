package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	configDir  = ".outpost"
	configFile = "config.json"
)

// Config holds the persistent CLI configuration stored at ~/.outpost/config.json.
type Config struct {
	AwsSecret         string    `json:"awsSecret"`
	AwsAccessKey      string    `json:"awsAccessKey"`
	AwsRegion         string    `json:"awsRegion"`
	AwsCredsUpdatedAt time.Time `json:"aws_creds_updated_at"`
	Mode              string    `json:"mode"`
}

// ConfigPath returns the absolute path to ~/.outpost/config.json.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads Config from ~/.outpost/config.json.
// If the file does not exist an empty Config is returned.
func Load() (*Config, error) {
	//backup.RestoreConfigIfNeeded()
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes cfg to ~/.outpost/config.json, creating the directory if needed.
func Save(cfg *Config) error {
	// backup.BeforeConfigSave()
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	configDir := filepath.Dir(path)
	_, statErr := os.Stat(configDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if os.IsNotExist(statErr) && runtime.GOOS == "darwin" {
		nosync := filepath.Join(configDir, ".nosync")
		_ = os.WriteFile(nosync, nil, 0600)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmpFile, err := os.CreateTemp(configDir, configFile+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
