// Package config loads git-snag's optional YAML configuration from the XDG
// config directory (~/.config/git-snag/config.yaml — global only, no
// per-project files; see ADR 0008).
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/kurabuchi-kentaro/git-snag/internal/scan"
)

// Config mirrors the YAML schema. Pointer fields distinguish "unset" from
// explicit zero values.
type Config struct {
	Scan struct {
		// Excludes are extra directory basenames merged on top of the
		// built-in exclude list.
		Excludes []string `yaml:"excludes"`
	} `yaml:"scan"`
	Animation struct {
		Enabled *bool `yaml:"enabled"`
	} `yaml:"animation"`
}

// DefaultPath returns the XDG-resolved config file location. XDG semantics
// are applied on every platform — including macOS, where os.UserConfigDir
// would ignore XDG_CONFIG_HOME and point at ~/Library/Application Support —
// so the documented path (~/.config/git-snag/config.yaml) holds everywhere.
func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "git-snag", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".config", "git-snag", "config.yaml"), nil
}

// Load reads the config file. A missing file is not an error — the tool is
// zero-config — but a present file that is unreadable or invalid fails
// loudly instead of silently falling back to defaults (REQ-A6).
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path) // #nosec G304 -- path is the user's own config location.
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg, nil
}

// Resolve combines the config with CLI flags into effective settings:
// the merged exclude set and whether the deletion animation plays.
// The --no-animation flag always wins over the config.
func Resolve(cfg Config, noAnimationFlag bool) (excludes map[string]bool, animation bool) {
	excludes = scan.DefaultExcludes()
	for _, name := range cfg.Scan.Excludes {
		excludes[name] = true
	}
	animation = true
	if cfg.Animation.Enabled != nil {
		animation = *cfg.Animation.Enabled
	}
	if noAnimationFlag {
		animation = false
	}
	return excludes, animation
}
