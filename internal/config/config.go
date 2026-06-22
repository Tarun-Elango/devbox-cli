package config

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"devbox-cli/internal/backup"
)

const (
	// fallbackServerURL is used when SERVER_URL is not found in the environment
	// or in a .env.local file. Primary configuration lives in .env.local.
	fallbackServerURL = "http://localhost:8080"
	configDir         = ".devbox"
	configFile        = "config.json"
)

// resolveServerURL returns the server URL from, in priority order:
//  1. SERVER_URL environment variable
//  2. SERVER_URL key in a .env.local file (searched by walking up from CWD)
//  3. Compiled-in fallbackServerURL
func resolveServerURL() string {
	if v := os.Getenv("SERVER_URL"); v != "" {
		return v
	}
	if path := findEnvFile(".env.local"); path != "" {
		if vars := parseDotEnv(path); vars["SERVER_URL"] != "" {
			return vars["SERVER_URL"]
		}
	}
	return fallbackServerURL
}

// findEnvFile walks up from the current working directory looking for a file
// with the given name. Returns its absolute path or "" if not found.
func findEnvFile(name string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// parseDotEnv reads a simple KEY=VALUE env file, ignoring blank lines and
// comments (lines starting with #). Surrounding quotes are stripped from values.
func parseDotEnv(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key != "" {
			vars[key] = val
		}
	}
	return vars
}

// Config holds the persistent CLI configuration stored at ~/.devbox/config.json.
type Config struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`
	ServerURL    string    `json:"serverUrl"`
	AwsSecret    string    `json:"awsSecret"`
	AwsAccessKey string    `json:"awsAccessKey"`
	AwsRegion    string    `json:"awsRegion"`
	Mode         string    `json:"mode"`
}

// IsTokenExpired reports whether the access token is expired or will expire
// within the next 30 seconds (buffer to avoid using a token right at its edge).
func (c *Config) IsTokenExpired() bool {
	if c.TokenExpiry.IsZero() {
		return true
	}
	return time.Now().After(c.TokenExpiry.Add(-30 * time.Second))
}

// configPath returns the absolute path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads Config from ~/.devbox/config.json.
// If the file does not exist an empty Config with the default server URL is returned.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{ServerURL: resolveServerURL()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = resolveServerURL()
	}
	return &cfg, nil // return config struct, and nil error
}

// Save writes cfg to ~/.devbox/config.json, creating the directory if needed.
func Save(cfg *Config) error {
	backup.BeforeConfigSave("local") // we dont care about cloud
	path, err := configPath()
	if err != nil {
		return err
	}
	configDir := filepath.Dir(path)
	_, statErr := os.Stat(configDir) // check if the directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if os.IsNotExist(statErr) && runtime.GOOS == "darwin" {
		nosync := filepath.Join(configDir, ".nosync") //if mac, and stat error, add the .nosync file to the directory
		_ = os.WriteFile(nosync, nil, 0600)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil { // only the owner can read/write
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Clear removes the saved tokens, effectively logging the user out.
func Clear() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Token = ""
	cfg.RefreshToken = ""
	cfg.TokenExpiry = time.Time{}
	return Save(cfg)
}

// ParseTokenExpiry decodes the exp claim from a JWT payload without verifying
// the signature. The token is assumed to be well-formed — it was just issued
// by a trusted server.
func ParseTokenExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}
