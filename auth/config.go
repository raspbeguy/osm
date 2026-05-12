package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// PersistedConfig holds CLI defaults written next to the token. Today it only
// remembers the OAuth client id so users don't have to set OSM_CLIENT_ID on
// every command.
type PersistedConfig struct {
	ClientID string `json:"client_id"`
}

func configPath() (string, error) {
	if p := os.Getenv("OSM_CONFIG_PATH"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(dir, "osm", "config.json"), nil
}

// LoadConfig reads the persisted config. Returns fs.ErrNotExist if absent.
func LoadConfig() (*PersistedConfig, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fs.ErrNotExist
		}
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	var c PersistedConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &c, nil
}

// SaveConfig writes the config with mode 0600, creating parents with 0700.
func SaveConfig(c *PersistedConfig) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(p), err)
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}
