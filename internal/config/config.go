// Package config loads the optional TOML file that sets the handover chain.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const defaultMaxAge = 6 * time.Hour

// Config is the handover policy loaded from disk.
type Config struct {
	Chain          []string
	Threshold      float64
	MaxAge         time.Duration
	AllowEstimated bool
	AllowStale     bool
}

type fileConfig struct {
	Chain          []string `toml:"chain"`
	Threshold      *float64 `toml:"threshold"`
	MaxAge         string   `toml:"max_age"`
	AllowEstimated *bool    `toml:"allow_estimated"`
	AllowStale     *bool    `toml:"allow_stale"`
}

// Default is the policy used when no config file exists.
func Default() Config {
	return Config{
		Chain: []string{
			"antigravity", "cursor", "claude", "codex", "hermes", "amp", "kiro",
		},
		Threshold:      15,
		MaxAge:         defaultMaxAge,
		AllowEstimated: false,
		AllowStale:     true,
	}
}

// DefaultPath is os.UserConfigDir()/baton/config.toml.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(dir, "baton", "config.toml"), nil
}

// Load reads path. An empty path uses BATON_CONFIG when set, otherwise
// DefaultPath. A missing default file returns Default. A missing explicit
// path, invalid TOML, or bad max_age is an error. Unknown keys are ignored.
func Load(path string) (Config, error) {
	explicit := path != ""
	if path == "" {
		if env := os.Getenv("BATON_CONFIG"); env != "" {
			path = env
			explicit = true
		} else {
			var err error
			path, err = DefaultPath()
			if err != nil {
				return Config{}, err
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var raw fileConfig
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg := Default()
	if raw.Chain != nil {
		cfg.Chain = append([]string(nil), raw.Chain...)
	}
	if raw.Threshold != nil {
		cfg.Threshold = *raw.Threshold
	}
	if raw.AllowEstimated != nil {
		cfg.AllowEstimated = *raw.AllowEstimated
	}
	if raw.AllowStale != nil {
		cfg.AllowStale = *raw.AllowStale
	}
	if raw.MaxAge != "" {
		d, err := time.ParseDuration(raw.MaxAge)
		if err != nil {
			return Config{}, fmt.Errorf("parse config max_age: %w", err)
		}
		cfg.MaxAge = d
	}
	return cfg, nil
}
