package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// SaveFileConfig writes cfg to the user's own otter.conf
// ($XDG_CONFIG_HOME/otter/otter.conf, or ~/.config/otter/otter.conf if
// XDG_CONFIG_HOME is unset), overwriting it entirely. It never writes to
// /etc/otter/otter.conf or /usr/share/otter/otter.conf, regardless of
// --root or where a currently-effective value happened to be sourced from
// — those are system-wide defaults, not something a per-user settings
// editor should touch.
//
// This always regenerates the whole file from cfg rather than patching it
// in place, so any comments or formatting-style choices (dotted keys,
// inline tables, table headers — see docs/05.configuration.md) already in
// the user's file are lost; the parsed settings themselves are preserved,
// which is what `otter settings` round-trips.
func SaveFileConfig(cfg *FileConfig) error {
	path := userConfigPath()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(*cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
