// Package config reads and writes the user's persistent preferences:
// ~/.config/stack-stitcher/config.yaml (or $XDG_CONFIG_HOME if set).
//
// The config file is a small YAML document:
//
//	theme: stitcher-dark
//
// More fields will land later (default file, keybinding overrides — see
// docs/ROADMAP.md). The struct and the write path are designed to absorb
// them without changing existing callers: Add a field, tag it, and
// LoadConfig/SaveConfig round-trip it automatically.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the persistent user preferences. Fields are exported for YAML
// marshalling; the zero value of each is the "not set" sentinel, and
// LoadConfig leaves a missing field at its zero rather than erroring.
type Config struct {
	// Theme is the registered theme name to activate on startup.
	// Empty means "use appstyles.DefaultTheme".
	Theme string `yaml:"theme,omitempty"`

	// URLHost overrides the host part of every service URL the app builds
	// (utils.URLHost). Empty means "detect it" - SSH_CONNECTION's server
	// address when running over SSH, "localhost" otherwise.
	URLHost string `yaml:"url_host,omitempty"`
}

// configDir returns the directory the config file lives in:
// $XDG_CONFIG_HOME/stack-stitcher if XDG_CONFIG_HOME is set, otherwise
// ~/.config/stack-stitcher.
func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "stack-stitcher"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "stack-stitcher"), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LoadConfig reads the config file and returns the parsed Config. A missing
// file is not an error: it returns the zero Config and a nil error, so the
// caller can apply defaults (DefaultTheme) without special-casing "first run".
// A malformed file is an error worth reporting — the caller decides whether
// to surface it or fall back to defaults.
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// SaveConfig writes cfg to the config file, creating the directory if
// needed. It is the whole persistence story for now: one call after a
// theme is chosen, one file, one write.
func SaveConfig(cfg Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0o644)
}
